package adapter

import "fmt"

func stripJSONC(input []byte) ([]byte, error) {
	output := make([]byte, 0, len(input))
	inString := false
	escaped := false
	for index := 0; index < len(input); index++ {
		current := input[index]
		if inString {
			output = append(output, current)
			if escaped {
				escaped = false
				continue
			}
			if current == '\\' {
				escaped = true
			} else if current == '"' {
				inString = false
			}
			continue
		}
		if current == '"' {
			inString = true
			output = append(output, current)
			continue
		}
		if current == '/' && index+1 < len(input) {
			next := input[index+1]
			if next == '/' {
				index += 2
				for index < len(input) && input[index] != '\n' {
					index++
				}
				if index < len(input) {
					output = append(output, '\n')
				}
				continue
			}
			if next == '*' {
				index += 2
				found := false
				for index+1 < len(input) {
					if input[index] == '*' && input[index+1] == '/' {
						index++
						found = true
						break
					}
					index++
				}
				if !found {
					return nil, fmt.Errorf("unterminated JSONC comment")
				}
				continue
			}
		}
		output = append(output, current)
	}
	if inString {
		return nil, fmt.Errorf("unterminated JSON string")
	}
	return removeTrailingCommas(output), nil
}

func removeTrailingCommas(input []byte) []byte {
	output := make([]byte, 0, len(input))
	inString := false
	escaped := false
	for index := 0; index < len(input); index++ {
		current := input[index]
		if inString {
			output = append(output, current)
			if escaped {
				escaped = false
				continue
			}
			if current == '\\' {
				escaped = true
			} else if current == '"' {
				inString = false
			}
			continue
		}
		if current == '"' {
			inString = true
			output = append(output, current)
			continue
		}
		if current != ',' {
			output = append(output, current)
			continue
		}
		next := index + 1
		for next < len(input) && (input[next] == ' ' || input[next] == '\t' || input[next] == '\n' || input[next] == '\r') {
			next++
		}
		if next < len(input) && (input[next] == '}' || input[next] == ']') {
			continue
		}
		output = append(output, input[index])
	}
	return output
}
