package core

import (
	"cloud.google.com/go/bigtable"
	"context"
	"time"
)

type Db struct {
	table        *bigtable.Table
	columnFamily string
}

func (self *Db) Init(tableName string, columnFamily string) {
	ctx := context.Background()
	var err error
	bigtable, err := bigtable.NewClient(ctx, "my-project-id", "my-instance")
	if err != nil {
		panic("Could not create data operations client: " + err.Error())
	}
	self.table = bigtable.Open(tableName)
	self.columnFamily = columnFamily
}

func (self *Db) Create(rowKey string, columnName string, value string, expiration time.Duration) (err error) {
	ctx, _ := context.WithTimeout(context.Background(), time.Second*2)
	mut := bigtable.NewMutation()
	t := time.Now().Add(expiration)
	timestamp := bigtable.Time(t)
	mut.Set(self.columnFamily, columnName, timestamp, []byte(value))
	err = self.table.Apply(ctx, rowKey, mut)
	return
}

func (self *Db) GetRows(rowKey string, opts ...bigtable.ReadOption) (data []string, err error) {
	var row bigtable.Row
	ctx, _ := context.WithTimeout(context.Background(), time.Second*2)
	row, err = self.table.ReadRow(ctx, rowKey,
		bigtable.RowFilter(
			bigtable.ChainFilters(
				bigtable.TimestampRangeFilter(time.Now(), time.Now().Add(time.Second*20)),
				bigtable.LatestNFilter(1))),
	)
	if err != nil {
		return
	}
	i := 0
	for _, r := range row {
		data = make([]string, len(r))
		for y, x := range r {
			data[y] = string(x.Value)
		}
		i++
	}
	return
}
