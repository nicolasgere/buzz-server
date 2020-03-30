package main

// [START bigtable_hw_imports]
import (
	"cloud.google.com/go/bigtable"
	"context"
	"log"
)

// [END bigtable_hw_imports]

// User-provided constants.
const (
	tableName        = "Hello-Bigtable"
	columnFamilyName = "cf1"
	columnName       = "greeting"
)

var greetings = []string{"Hello World!", "Hello Cloud Bigtable!", "Hello golang!"}

// sliceContains reports whether the provided string is present in the given slice of strings.
func sliceContains(list []string, target string) bool {
	for _, s := range list {
		if s == target {
			return true
		}
	}
	return false
}

func main() {
	ctx := context.Background()
	adminClient, err := bigtable.NewAdminClient(ctx, "my-project-id", "my-instance")
	if err != nil {
		log.Fatalf("Could not create admin client: %v", err)
	}
	tableName := "buzz"
	columnFamilyName := "apt"

	if err := adminClient.CreateTable(ctx, tableName); err != nil {
			log.Fatalf("Could not create table %s: %v", tableName, err)
	}

	if err := adminClient.CreateColumnFamily(ctx, tableName, columnFamilyName); err != nil {
			log.Fatalf("Could not create column family %s: %v", columnFamilyName, err)}
}