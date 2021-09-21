// package store is a simple, in-memory key-value store
package store

import "fmt"

type Store struct {
	data map[string]string
}

func New() *Store {
	return &Store{
		data: make(map[string]string),
	}
}

func (s *Store) Set(key, val string) {
	s.data[key] = val
}

func (s *Store) Get(key string) string {
	return s.data[key]
}

func (s *Store) String() string {
	result := ""
	for k, v := range s.data {
		result += fmt.Sprintf("%v: %v\n", k, v)
	}
	return result
}
