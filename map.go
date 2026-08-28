package equix

import "fmt"

func mapVerify(code int) error {
	switch code {
	case 0:
		return nil
	case 1:
		return ErrChallenge
	case 2:
		return ErrOrder
	case 3:
		return ErrPartialSum
	case 4:
		return ErrFinalSum
	default:
		return fmt.Errorf("equix: unknown result %d", code)
	}
}

func toSolutions(in [][8]uint16) []Solution {
	out := make([]Solution, len(in))
	for i := range in {
		out[i] = Solution(in[i])
	}
	return out
}
