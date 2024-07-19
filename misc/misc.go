package misc

// InSlice checks to see if a value is in a given slice
func InSlice[T comparable](val T, list []T) bool {
	for _, key := range list {
		if key == val {
			return true
		}
	}
	return false
}

// UniqueSlice takes a slice and removes duplicates
func UniqueSlice[T comparable](slice []T) []T {
	keys := make(map[T]bool)
	list := []T{}
	for _, entry := range slice {
		if _, value := keys[entry]; !value {
			keys[entry] = true
			list = append(list, entry)
		}
	}
	return list
}

func FlattenSlice[T comparable](slice [][]T) []T {
	flat := []T{}
	for _, sub := range slice {
		flat = append(flat, sub...)
	}
	return flat
}

func FlattenMap[T comparable, U any](m map[T][]U) []U {
	flat := []U{}
	for _, sub := range m {
		flat = append(flat, sub...)
	}
	return flat
}

func SliceOverlap[T comparable](a, b []T) bool {
	for _, v := range a {
		if InSlice(v, b) {
			return true
		}
	}
	return false
}

// SliceEqual checks if two slices are equal
func SliceEqual[T comparable](a, b []T) bool {
	if len(a) != len(b) {
		return false
	}
	for i, v := range a {
		if v != b[i] {
			return false
		}
	}
	return true
}

func ToInterfaceSlice[T any](arr []T) []interface{} {
	items := []interface{}{}
	for _, item := range arr {
		items = append(items, item)
	}
	return items
}

// func Pointer[T constraints.Ordered](v T) *T {
// 	return &v
// }

func Pointer[T any](v T) *T {
	return &v
}

func PointerBool(v bool) *bool {
	return &v
}
