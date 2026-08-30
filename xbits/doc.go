// Package xbits 哈希前缀难度封装。
// Equi-X v2 版本裸运算求解难度约 2.5ms，此封装为难度约 80ms ~ 100ms。
//
// 用法：
//
//	func main() {
//		challenge := []byte("example_challenge_string")
//
//		fmt.Println("Solving puzzle...")
//		sol, err := SolvePuzzle(challenge)
//		if err != nil {
//			panic(err)
//		}
//
//		fmt.Printf("Found Solution! Nonce: %d\n", sol.Nonce)
//
//		// 服务端快速验证
//		valid := VerifyPuzzle(challenge, sol)
//		fmt.Printf("Verification result: %v\n", valid)
//	}
package xbits
