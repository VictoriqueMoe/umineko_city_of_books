package utils

import "github.com/google/uuid"

type treeNode[T any] struct {
	data     T
	children []*treeNode[T]
}

func BuildTree[T any](
	flat []T,
	getID func(T) uuid.UUID,
	getParentID func(T) *uuid.UUID,
	setReplies func(*T, []T),
) []T {
	nodes := make(map[uuid.UUID]*treeNode[T], len(flat))
	for i := range flat {
		nodes[getID(flat[i])] = &treeNode[T]{data: flat[i]}
	}

	var roots []*treeNode[T]
	for i := range flat {
		parentID := getParentID(flat[i])
		if parentID == nil {
			roots = append(roots, nodes[getID(flat[i])])
		} else {
			if parent, ok := nodes[*parentID]; ok {
				parent.children = append(parent.children, nodes[getID(flat[i])])
			} else {
				roots = append(roots, nodes[getID(flat[i])])
			}
		}
	}

	result := make([]T, len(roots))
	for i, r := range roots {
		result[i] = flatten(r, setReplies)
	}
	return result
}

func flatten[T any](n *treeNode[T], setReplies func(*T, []T)) T {
	resp := n.data
	var replies []T
	for _, child := range n.children {
		replies = append(replies, flatten(child, setReplies))
	}
	setReplies(&resp, replies)
	return resp
}
