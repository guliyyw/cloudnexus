package snowflake

import (
	"log"
	"sync"

	sf "github.com/bwmarrin/snowflake"
)

var (
	node *sf.Node
	mu   sync.Mutex
)

func Init(nodeID int64) {
	var err error
	node, err = sf.NewNode(nodeID)
	if err != nil {
		log.Fatalf("snowflake init failed: %v", err)
	}
	log.Printf("snowflake node %d initialized", nodeID)
}

func Generate() sf.ID {
	mu.Lock()
	defer mu.Unlock()
	return node.Generate()
}

func Uint64() uint64 {
	return uint64(Generate())
}
