// The demo command was automatically generated by Shenzhen Go.
package main

import (
	"fmt"
	"sync"
)

func Node_1(qux chan<- int) {

	func(instanceNumber, multiplicity int) {
		fmt.Println("Node 1: Started.")
		fmt.Print("Enter a number: ")
		var n int
		fmt.Scanf("%d", &n)
		fmt.Printf("Node 1: Sending %d on qux...\n", n)
		qux <- n
		fmt.Println("Node 1: Finished.")
	}(0, 1)

}

func Node_2(foo <-chan int) {

	func(instanceNumber, multiplicity int) {
		fmt.Println("Node 2: Started.")
		fmt.Println("Node 2: Waiting on foo...")
		fmt.Printf("Node 2: Got %d on foo\n", <-foo)
		fmt.Println("Node 2: Finished.")
	}(0, 1)

}

func main() {

	channel0 := make(chan int, 0)

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		Node_1(channel0)
		wg.Done()
	}()

	wg.Add(1)
	go func() {
		Node_2(channel0)
		wg.Done()
	}()

	// Wait for the end
	wg.Wait()
}
