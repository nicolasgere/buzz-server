package main

// [START bigtable_hw_imports]
import (
	"cloud.google.com/go/bigtable"
	"context"
	"fmt"
	"log"
	"time"
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
	tableBuzz := "buzz"
	columnFamilyName := "apt"

	if err := adminClient.CreateTable(ctx, tableBuzz); err != nil {
		fmt.Printf("Could not create table %s: %v \n", tableBuzz, err)
	}

	if err := adminClient.CreateColumnFamily(ctx, tableBuzz, columnFamilyName); err != nil {
		fmt.Printf("Could not create column family %s: %v \n", columnFamilyName, err)}

	tableHeartbeat := "heartbeat"
	columnFamilyNameBear := "beat"

	if err := adminClient.CreateTable(ctx, tableHeartbeat); err != nil {
		fmt.Printf("Could not create table %s: %v \n", tableName, err)
	}

	if err := adminClient.CreateColumnFamily(ctx, tableHeartbeat, columnFamilyNameBear); err != nil {
		fmt.Printf("Could not create column family %s: %v \n", columnFamilyNameBear, err)}

	// Set a garbage collection policy of 5 days.
	maxAge := time.Second * 1
	policy := bigtable.MaxAgePolicy(maxAge)
	if err := adminClient.SetGCPolicy(ctx, tableHeartbeat, columnFamilyNameBear, policy); err != nil {
		 fmt.Errorf("Could not SetGCPolicy(%s): %v \n", policy, err)
	}

	policy2 := bigtable.MaxVersionsPolicy(1)
	if err := adminClient.SetGCPolicy(ctx, tableHeartbeat, columnFamilyNameBear, policy2); err != nil {
		fmt.Errorf("Could not SetGCPolicy(%s): %v \n", policy, err)
	}

	fmt.Println("DONE")
}