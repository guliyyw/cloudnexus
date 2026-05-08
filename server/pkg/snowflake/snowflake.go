package snowflake

import (
	"log"
	"os"
	"strconv"
	"sync"

	sf "github.com/bwmarrin/snowflake"
)

var (
	node *sf.Node
	mu   sync.Mutex
)

// Init initializes the snowflake node. If the SNOWFLAKE_NODE_ID env var is set,
// it overrides the defaultID. The node ID must be in [0, 1023].
func Init(defaultID int64) {
	nodeID := defaultID
	if v := os.Getenv("SNOWFLAKE_NODE_ID"); v != "" {
		if id, err := strconv.ParseInt(v, 10, 64); err == nil {
			nodeID = id
		}
	}
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
