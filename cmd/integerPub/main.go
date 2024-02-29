package main

import (
	"fmt"

	"iter"

)

//
// func main() {
// 	publisher := rivulet.NewEventBridgePublisher()
// 	for i := range 5 {
// 		msg := fmt.Sprintf("Todays lucky number is %d\n", i)
// 		err := publisher.Publish(msg)
// 		if err != nil {
// 			fmt.Println(err)
// 			os.Exit(1)
// 		}
// 	}
// }
//
//
//

func main() {
	for i, v := range integers() {
		if i == 3 {
			break
		}
		fmt.Println(i)
		fmt.Println(v)
		fmt.Println()
	}
}

func integers() iter.Seq2[int, int] {
	return func(yield func(int, int) bool) {
		for i := range 5 {
			if !yield(i , i  +1) {
				break
			}
		}
	}
}
