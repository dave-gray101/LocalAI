package utils

// TODO: noramlly, these go in utils_tests, right? Why does this cause problems only in pkg/utils?

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	// "github.com/rs/zerolog/log"
)

var _ = Describe("utils/concurrency tests", func() {
	It("SliceOfChannelsReducer works", func() {
		individualResultsChannels := []<-chan int{}
		initialValue := 0
		for i := 0; i < 3; i++ {
			c := make(chan int)
			go func(i int, c chan int) {
				for ii := 1; ii < 4; ii++ {
					c <- (i * ii)
				}
				close(c)
			}(i, c)
			individualResultsChannels = append(individualResultsChannels, c)
		}
		Expect(len(individualResultsChannels)).To(Equal(3))
		finalResultChannel := make(chan int)
		wg := SliceOfChannelsReducer[int, int](individualResultsChannels, finalResultChannel, func(input int, val int) int {
			return val + input
		}, initialValue, true)

		Expect(wg).ToNot(BeNil())

		result := <-finalResultChannel

		Expect(result).ToNot(Equal(0))
		Expect(result).To(Equal(18))
	})
})