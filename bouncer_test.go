package bouncer

import "testing"

func TestBind(t *testing.T) {
	for _, testCase := range jsonTestCases {
		performJsonTest(t, testCase)
	}
}
