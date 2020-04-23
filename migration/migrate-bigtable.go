package migration

import (
	"cloud.google.com/go/bigtable"
	"context"
	"fmt"
	"log"
	"time"
)

func Migrate() {
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
		fmt.Printf("Could not create table %s: %v \n", tableHeartbeat, err)
	}

	if err := adminClient.CreateColumnFamily(ctx, tableHeartbeat, columnFamilyNameBear); err != nil {
		fmt.Printf("Could not create column family %s: %v \n", columnFamilyNameBear, err)}

	// Set a garbage collection policy of 5 days.
	maxAge := time.Second * 1
	policy := bigtable.MaxAgePolicy(maxAge)
	if err := adminClient.SetGCPolicy(ctx, tableHeartbeat, columnFamilyNameBear, policy); err != nil {
		 fmt.Errorf("Could not SetGCPolicy(%s): %v \n", policy, err)
	}
	if err := adminClient.SetGCPolicy(ctx, tableBuzz, columnFamilyName, policy); err != nil {
		fmt.Errorf("Could not SetGCPolicy(%s): %v \n", policy, err)
	}
	fmt.Println("Migration done")
}