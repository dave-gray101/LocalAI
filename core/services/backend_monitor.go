package services

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-skynet/LocalAI/core/config"
	"github.com/go-skynet/LocalAI/core/schema"
	"github.com/go-skynet/LocalAI/pkg/grpc/proto"
	"github.com/go-skynet/LocalAI/pkg/model"

	"github.com/rs/zerolog/log"

	gopsutil "github.com/shirou/gopsutil/v3/process"
)

// This utility extension is used for backend_monitor and backend_rules, but nowhere outside of service.
// trying out this style - TODO is it better or worse
func getModelLoaderIDFromModelName(bcl *config.BackendConfigLoader, modelName string) (string, config.BackendConfig, error) {
	config, exists := bcl.GetBackendConfig(modelName)
	var backendId string
	if exists {
		backendId = config.Model
	} else {
		// Last ditch effort: use it raw, see if a backend happens to match.
		backendId = modelName
	}

	if !strings.HasSuffix(backendId, ".bin") {
		backendId = fmt.Sprintf("%s.bin", backendId)
	}

	return backendId, config, nil
}

type BackendMonitor struct {
	configLoader *config.BackendConfigLoader
	modelLoader  *model.ModelLoader
	options      *config.ApplicationConfig // Taking options in case we need to inspect ExternalGRPCBackends, though that's out of scope for now, hence the name.
}

func NewBackendMonitor(configLoader *config.BackendConfigLoader, modelLoader *model.ModelLoader, appConfig *config.ApplicationConfig) BackendMonitor {
	return BackendMonitor{
		configLoader: configLoader,
		modelLoader:  modelLoader,
		options:      appConfig,
	}
}

func (bm *BackendMonitor) SampleLocalBackendProcess(model string) (*schema.BackendMonitorResponse, error) {
	config, exists := bm.configLoader.GetBackendConfig(model)
	var backend string
	if exists {
		backend = config.Model
	} else {
		// Last ditch effort: use it raw, see if a backend happens to match.
		backend = model
	}

	if !strings.HasSuffix(backend, ".bin") {
		backend = fmt.Sprintf("%s.bin", backend)
	}

	pid, err := bm.modelLoader.GetGRPCPID(backend)

	if err != nil {
		log.Error().Msgf("model %s : failed to find pid %+v", model, err)
		return nil, err
	}

	// Name is slightly frightening but this does _not_ create a new process, rather it looks up an existing process by PID.
	backendProcess, err := gopsutil.NewProcess(int32(pid))

	if err != nil {
		log.Error().Msgf("model %s [PID %d] : error getting process info %+v", model, pid, err)
		return nil, err
	}

	memInfo, err := backendProcess.MemoryInfo()

	if err != nil {
		log.Error().Msgf("model %s [PID %d] : error getting memory info %+v", model, pid, err)
		return nil, err
	}

	memPercent, err := backendProcess.MemoryPercent()
	if err != nil {
		log.Error().Msgf("model %s [PID %d] : error getting memory percent %+v", model, pid, err)
		return nil, err
	}

	cpuPercent, err := backendProcess.CPUPercent()
	if err != nil {
		log.Error().Msgf("model %s [PID %d] : error getting cpu percent %+v", model, pid, err)
		return nil, err
	}

	return &schema.BackendMonitorResponse{
		MemoryInfo:    memInfo,
		MemoryPercent: memPercent,
		CPUPercent:    cpuPercent,
	}, nil
}

func (bm BackendMonitor) CheckAndSample(modelName string) (*proto.StatusResponse, error) {
	backendId, _, err := getModelLoaderIDFromModelName(bm.configLoader, modelName)
	if err != nil {
		return nil, err
	}
	lmm := bm.modelLoader.CheckIsLoaded(backendId, false)
	if lmm.ModelAddress == "" {
		return nil, fmt.Errorf("backend %s is not currently loaded", backendId)
	}

	status, rpcErr := lmm.ModelAddress.GRPC(false, nil).Status(context.TODO())
	if rpcErr != nil {
		log.Warn().Msgf("backend %s experienced an error retrieving status info: %s", backendId, rpcErr.Error())
		val, slbErr := bm.SampleLocalBackendProcess(backendId)
		if slbErr != nil {
			return nil, fmt.Errorf("backend %s experienced an error retrieving status info via rpc: %s, then failed local node process sample: %s", backendId, rpcErr.Error(), slbErr.Error())
		}
		return &proto.StatusResponse{
			State: proto.StatusResponse_ERROR,
			Memory: &proto.MemoryUsageData{
				Total: val.MemoryInfo.VMS,
				Breakdown: map[string]uint64{
					"gopsutil-RSS": val.MemoryInfo.RSS,
				},
			},
		}, nil
	}
	return status, nil
}

func (bm BackendMonitor) ShutdownModel(modelName string) error {
	backendId, _, err := getModelLoaderIDFromModelName(bm.configLoader, modelName)
	if err != nil {
		return err
	}
	return bm.modelLoader.ShutdownModel(backendId)
}
