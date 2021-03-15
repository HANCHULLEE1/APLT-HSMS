package ast

import (
	"fmt"
	"strconv"
	"strings"
)

// UintNode is a immutable data type that represents an unsigned integer in a SECS-II message.
// Implements DataItemNode.
type UintNode struct {
	byteSize  int            // Byte size of the unsigned integers; should be either 1, 2, 4, or 8
	values    []uint64       // Array of unsigned integers
	variables map[string]int // Variable name and its position in the data array

	// Rep invariants
	// - Each values[i] should be in range of [0, max], where max = 1<<(byteSize*8)-1
	// - If a variable exists in position i, values[i] will be zero-value (0) and should not be used.
	// - variable name should adhere to the variable naming rule; refer to interface.go
	// - variable positions should be unique, and be in range of [0, len(values))
}

// Factory methods

// NewUintNode creates a new UintNode that contains unsigned integer data.
//
// The byteSize should be either 1, 2, 4, or 8.
// Each input of the values should be an unsigned integer that could be represented within bytes of the byteSize,
// or it should be a string with a valid variable name as specified in the interface documentation.
func NewUintNode(byteSize int, values ...interface{}) ItemNode {
	if getDataByteLength(fmt.Sprintf("u%d", byteSize), len(values)) > MAX_BYTE_SIZE {
		panic("item node size limit exceeded")
	}

	var (
		nodeValues    []uint64       = make([]uint64, 0, len(values))
		nodeVariables map[string]int = make(map[string]int)
	)

	for i, value := range values {
		switch value.(type) {
		case int:
			nodeValues = append(nodeValues, uint64(value.(int)))
		case int8:
			nodeValues = append(nodeValues, uint64(value.(int8)))
		case int16:
			nodeValues = append(nodeValues, uint64(value.(int16)))
		case int32:
			nodeValues = append(nodeValues, uint64(value.(int32)))
		case int64:
			nodeValues = append(nodeValues, uint64(value.(int64)))
		case uint:
			nodeValues = append(nodeValues, uint64(value.(uint)))
		case uint8:
			nodeValues = append(nodeValues, uint64(value.(uint8)))
		case uint16:
			nodeValues = append(nodeValues, uint64(value.(uint16)))
		case uint32:
			nodeValues = append(nodeValues, uint64(value.(uint32)))
		case uint64:
			nodeValues = append(nodeValues, value.(uint64))
		case string:
			v := value.(string)
			nodeValues = append(nodeValues, 0)
			nodeVariables[v] = i
		default:
			panic("input argument contains invalid type for UintNode")
		}
	}

	node := &UintNode{byteSize, nodeValues, nodeVariables}
	node.checkRep()
	return node
}

// Public methods

// Size implements ItemNode.Size().
func (node *UintNode) Size() int {
	return len(node.values)
}

// Variables implements ItemNode.Variables().
func (node *UintNode) Variables() []string {
	return getVariableNames(node.variables)
}

// FillValues implements ItemNode.FillValues().
func (node *UintNode) FillValues(values map[string]interface{}) ItemNode {
	nodeValues := make([]interface{}, 0, node.Size())
	for _, v := range node.values {
		nodeValues = append(nodeValues, v)
	}
	for name, pos := range node.variables {
		if v, ok := values[name]; ok {
			nodeValues[pos] = v
		} else {
			nodeValues[pos] = name
		}
	}
	return NewUintNode(node.byteSize, nodeValues...)
}

// ToBytes implements ItemNode.ToBytes()
func (node *UintNode) ToBytes() []byte {
	if len(node.variables) != 0 {
		return []byte{}
	}

	result, err := getHeaderBytes(fmt.Sprintf("u%d", node.byteSize), node.Size())
	if err != nil {
		return []byte{}
	}

	for _, value := range node.values {
		// Initialize mask; mask == 0xFF000000 when node.byteSize == 4
		var mask uint64 = 0xFF << ((node.byteSize - 1) * 8)
		for i := 0; i < node.byteSize; i++ {
			// Calculate and append value's i-th byte
			// e.g. given value == 0x01ABCDEF, node.ByteSize == 4,
			//      ithByte == 0x01 when i == 0
			//      ithByte == 0xAB when i == 1
			var ithByte byte = byte((value & mask) >> ((node.byteSize - i - 1) * 8))
			result = append(result, ithByte)
			mask = mask >> 8
		}
	}

	return result
}

// String returns the string representation of the node.
func (node *UintNode) String() string {
	if node.Size() == 0 {
		return fmt.Sprintf("<U%d[0]>", node.byteSize)
	}

	values := make([]string, 0, node.Size())
	for _, v := range node.values {
		values = append(values, strconv.FormatUint(v, 10))
	}

	for name, pos := range node.variables {
		values[pos] = name
	}

	return fmt.Sprintf("<U%d[%d] %v>", node.byteSize, node.Size(), strings.Join(values, " "))
}

// Private methods

func (node *UintNode) checkRep() {
	if node.byteSize != 1 && node.byteSize != 2 &&
		node.byteSize != 4 && node.byteSize != 8 {
		panic("invalid byte size")
	}

	for _, v := range node.values {
		if !(0 <= v && v <= uint64(1<<(node.byteSize*8)-1)) {
			panic("value overflow")
		}
	}

	visited := map[int]bool{}
	for name, pos := range node.variables {
		if node.values[pos] != 0 {
			panic("value in variable position isn't a zero-value")
		}

		if !isValidVarName(name) {
			panic("invalid variable name")
		}

		if _, ok := visited[pos]; ok {
			panic("variable position is not unique")
		}
		visited[pos] = true

		if !(0 <= pos && pos < node.Size()) {
			panic("variable position overflow")
		}
	}
}