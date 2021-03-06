package cube

import (
	"github.com/bahadrix/cardinalitycube/cube/pb"
	"sync"
)

// Board is a table like data structure which consists of rows.
// It is of course thread safe.
type Board struct {
	cube     *Cube
	rowMap   map[string]*Row
	rowLock  sync.RWMutex
	cellLock sync.Mutex
}

// A BoardSnapshot contains rows data of specific time
type BoardSnapshot map[string]*RowSnapshot

// NewBoard creates a new board for given cube
func NewBoard(cube *Cube) *Board {
	return &Board{
		cube:   cube,
		rowMap: make(map[string]*Row),
	}
}

// GetCell returns cell that resides in given row.
// If row or cell not found function returns nil
func (b *Board) GetCell(rowName string, cellName string, createIfNotExists bool) *Cell {

	var cell *Cell
	b.rowLock.RLock()
	row, _ := b.rowMap[rowName]
	b.rowLock.RUnlock()

	if row == nil {
		if !createIfNotExists {
			return nil
		}
		b.rowLock.Lock() // row sync in ----
		row, _ = b.rowMap[rowName]
		if row == nil {
			row = NewRow()
			b.rowMap[rowName] = row
		}
		b.rowLock.Unlock() // row sync out ---
	}

	cell = row.GetCell(cellName)

	if cell == nil && createIfNotExists {
		b.cellLock.Lock() // cell sync in ----
		cell = row.GetCell(cellName)
		if cell == nil {
			cell = b.cube.generateCell()
			row.SetCell(cellName, cell)
		}
		b.cellLock.Unlock() // cell sync out ---
	}

	return cell
}

// GetRowSnapshot Returns snapshot of given row.
// Blocks row while getting its snapshot
func (b *Board) GetRowSnapshot(rowName string) *RowSnapshot {
	b.rowLock.RLock()
	row, _ := b.rowMap[rowName]
	b.rowLock.RUnlock()

	if row == nil {
		return nil
	}
	return row.GetSnapshot()
}

// GetSnapshot return board's snapshot.
// Blocks whole board while getting snapshot.
func (b *Board) GetSnapshot() *BoardSnapshot {
	ss := make(BoardSnapshot)
	b.rowLock.RLock()
	for key, row := range b.rowMap {
		ss[key] = row.GetSnapshot()
	}
	b.rowLock.RUnlock()

	return &ss
}

// CheckRowExists return true if row exists in board.
func (b *Board) CheckRowExists(rowName string) bool {
	b.rowLock.RLock()
	_, exists := b.rowMap[rowName]
	b.rowLock.RUnlock()
	return exists
}

// DropRow drops given row from board if it exists
func (b *Board) DropRow(rowName string) {
	b.rowLock.Lock()
	_, rowExists := b.rowMap[rowName]
	if rowExists {
		delete(b.rowMap, rowName)
	}
	b.rowLock.Unlock()
}

// GetRowKeys returns row names. Read blocking operation.
func (b *Board) GetRowKeys() []string {
	b.rowLock.RLock()
	keys := make([]string, 0, len(b.rowMap))
	for key := range b.rowMap {
		keys = append(keys, key)
	}
	b.rowLock.RUnlock()

	return keys

}

// GetRowCount returns roe count.
func (b *Board) GetRowCount() int {
	return len(b.rowMap)
}

// GetCellKeys returns cell keys of row. Read blocking operation.
func (b *Board) GetCellKeys(rowName string) (keys []string) {
	b.rowLock.RLock()
	r, rowExists := b.rowMap[rowName]
	b.rowLock.RUnlock()
	if !rowExists {
		return
	}
	return r.GetCellKeys()
}

// GetCellCount returns cell count of row. Read blocking operation.
func (b *Board) GetCellCount(rowName string) int {
	b.rowLock.RLock()
	r, rowExists := b.rowMap[rowName]
	b.rowLock.RUnlock()
	if !rowExists {
		return 0
	}
	return r.GetCellCount()
}


func (b *Board) Dump() (*pb.BoardData, error) {
	b.rowLock.RLock()
	defer b.rowLock.RUnlock()

	dataMap := make(map[string]*pb.RowData, len(b.rowMap))
	var err error
	for k, r := range b.rowMap {
		dataMap[k], err = r.Dump()

		if err != nil {
			return nil, err
		}

	}

	return &pb.BoardData{RowMap:dataMap}, err
}

func (b *Board) LoadData(data *pb.BoardData) error {

	b.rowLock.Lock()
	defer b.rowLock.Unlock()

	for rowName, rowData := range data.RowMap {

		row, rowExists := b.rowMap[rowName]

		if !rowExists {
			row = NewRow()
			b.rowMap[rowName] = row
		}

		for cellName, cellData := range rowData.CellMap {
			cell, err := b.cube.deserializeCell(cellData.CoreData)
			if err != nil {
				return err
			}
			row.SetCell(cellName, cell)
		}
	}

	return nil

}


