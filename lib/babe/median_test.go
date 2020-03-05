package babe

import (
	"math/big"
	"testing"
	"time"

	"github.com/ChainSafe/gossamer/dot/core/types"
)

func TestMedian_OddLength(t *testing.T) {
	us := []uint64{3, 2, 1, 4, 5}
	res, err := median(us)
	if err != nil {
		t.Fatal(err)
	}

	var expected uint64 = 3

	if res != expected {
		t.Errorf("Fail: got %v expected %v\n", res, expected)
	}

}

func TestMedian_EvenLength(t *testing.T) {
	us := []uint64{1, 4, 2, 4, 5, 6}
	res, err := median(us)
	if err != nil {
		t.Fatal(err)
	}

	var expected uint64 = 4

	if res != expected {
		t.Errorf("Fail: got %v expected %v\n", res, expected)
	}

}

func TestSlotOffset_Failing(t *testing.T) {
	var st uint64 = 1000001
	var se uint64 = 1000000

	_, err := slotOffset(st, se)
	if err == nil {
		t.Fatal("Fail: did not err for c>1")
	}

}

func TestSlotOffset(t *testing.T) {
	var st uint64 = 1000000
	var se uint64 = 1000001

	res, err := slotOffset(st, se)
	if err != nil {
		t.Fatal(err)
	}

	var expected uint64 = 1

	if res != expected {
		t.Errorf("Fail: got %v expected %v\n", res, expected)
	}
}

func addBlocksToState(t *testing.T, babesession *Session, depth int, blockState BlockState, startTime uint64) {
	previousHash := blockState.BestBlockHash()
	previousAT := startTime

	for i := 1; i <= depth; i++ {

		// create proof that we can authorize this block
		babesession.epochThreshold = big.NewInt(0)
		babesession.authorityIndex = 0
		slotNumber := uint64(i)

		outAndProof, err := babesession.runLottery(slotNumber)
		if err != nil {
			t.Fatal(err)
		}

		if outAndProof == nil {
			t.Fatal("proof was nil when over threshold")
		}

		babesession.slotToProof[slotNumber] = outAndProof

		// create pre-digest
		slot := Slot{
			start:    uint64(time.Now().Unix()),
			duration: uint64(1000),
			number:   slotNumber,
		}

		predigest, err := babesession.buildBlockPreDigest(slot)
		if err != nil {
			t.Fatal(err)
		}

		block := &types.Block{
			Header: &types.Header{
				ParentHash: previousHash,
				Number:     big.NewInt(int64(i)),
				Digest:     [][]byte{predigest.Encode()},
			},
			Body: &types.Body{},
		}

		arrivalTime := previousAT + uint64(1)
		previousHash = block.Header.Hash()
		previousAT = arrivalTime

		err = blockState.AddBlockWithArrivalTime(block, arrivalTime)
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestSlotTime(t *testing.T) {
	babesession, dbSrv := createTestSessionWithState(t, nil)
	defer func() {
		err := dbSrv.Stop()
		if err != nil {
			t.Fatal(err)
		}
	}()

	addBlocksToState(t, babesession, 100, dbSrv.Block, uint64(0))

	res, err := babesession.slotTime(103, 20)
	if err != nil {
		t.Fatal(err)
	}

	expected := uint64(103)

	if res != expected {
		t.Errorf("Fail: got %v expected %v\n", res, expected)
	}
}

func TestEstimateCurrentSlot(t *testing.T) {
	babesession, dbSrv := createTestSessionWithState(t, nil)

	defer func() {
		err := dbSrv.Stop()
		if err != nil {
			t.Fatal(err)
		}
	}()

	// create proof that we can authorize this block
	babesession.epochThreshold = big.NewInt(0)
	babesession.authorityIndex = 0
	slotNumber := uint64(17)

	outAndProof, err := babesession.runLottery(slotNumber)
	if err != nil {
		t.Fatal(err)
	}

	if outAndProof == nil {
		t.Fatal("proof was nil when over threshold")
	}

	babesession.slotToProof[slotNumber] = outAndProof

	// create pre-digest
	slot := Slot{
		start:    uint64(time.Now().Unix()),
		duration: babesession.config.SlotDuration,
		number:   slotNumber,
	}

	predigest, err := babesession.buildBlockPreDigest(slot)
	if err != nil {
		t.Fatal(err)
	}

	block := &types.Block{
		Header: &types.Header{
			ParentHash: genesisHeader.Hash(),
			Number:     big.NewInt(int64(1)),
			Digest:     [][]byte{predigest.Encode()},
		},
		Body: &types.Body{},
	}

	arrivalTime := uint64(time.Now().Unix()) - slot.duration

	err = dbSrv.Block.AddBlockWithArrivalTime(block, arrivalTime)
	if err != nil {
		t.Fatal(err)
	}

	estimatedSlot, err := babesession.estimateCurrentSlot()
	if err != nil {
		t.Fatal(err)
	}

	if estimatedSlot != slotNumber+1 {
		t.Fatalf("Fail: got %d expected %d", estimatedSlot, slotNumber+1)
	}
}

func TestGetCurrentSlot(t *testing.T) {
	babesession, dbSrv := createTestSessionWithState(t, nil)

	defer func() {
		err := dbSrv.Stop()
		if err != nil {
			t.Fatal(err)
		}
	}()

	// 100 blocks / 1000 ms/s
	// TODO: use time.Duration
	addBlocksToState(t, babesession, 100, dbSrv.Block, uint64(time.Now().Unix())-(babesession.config.SlotDuration/10))

	res, err := babesession.getCurrentSlot()
	if err != nil {
		t.Fatal(err)
	}

	expected := uint64(101)

	if res != expected && res != expected+1 {
		t.Fatalf("Fail: got %d expected %d", res, expected)
	}
}