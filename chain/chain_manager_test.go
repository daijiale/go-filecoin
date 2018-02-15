package chain

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	hamt "gx/ipfs/QmdBXcN47jVwKLwSyN9e9xYVZ7WcAWgQ5N4cmNw7nzWq2q/go-hamt-ipld"

	types "github.com/filecoin-project/go-filecoin/types"
)

var (
	testGenesis = &types.Block{}

	block1 = &types.Block{
		Parent: testGenesis.Cid(),
		Height: 1,
	}

	block2 = &types.Block{
		Parent: block1.Cid(),
		Height: 2,
	}

	fork1 = &types.Block{
		Parent: testGenesis.Cid(),
		Height: 1,
		Nonce:  1,
	}

	fork2 = &types.Block{
		Parent: fork1.Cid(),
		Height: 2,
	}

	fork3 = &types.Block{
		Parent: fork2.Cid(),
		Height: 3,
	}
)

func TestBasicAddBlock(t *testing.T) {
	assert := assert.New(t)
	ctx := context.Background()
	cs := hamt.NewCborStore()
	stm := NewChainManager(cs)

	assert.NoError(stm.SetBestBlock(ctx, testGenesis))

	res, err := stm.ProcessNewBlock(ctx, block1)
	assert.NoError(err)
	assert.Equal(ChainAccepted, res)
	assert.Equal(stm.bestBlock.blk.Cid(), block1.Cid())
	assert.True(stm.KnownGoodBlocks.Has(block1.Cid()))

	res, err = stm.ProcessNewBlock(ctx, block2)
	assert.NoError(err)
	assert.Equal(ChainAccepted, res)
	assert.Equal(stm.bestBlock.blk.Cid(), block2.Cid())
	assert.True(stm.KnownGoodBlocks.Has(block2.Cid()))
}

func TestForkChoice(t *testing.T) {
	assert := assert.New(t)
	ctx := context.Background()
	cs := hamt.NewCborStore()
	stm := NewChainManager(cs)

	assert.NoError(stm.SetBestBlock(ctx, testGenesis))

	res, err := stm.ProcessNewBlock(ctx, block1)
	assert.NoError(err)
	assert.Equal(ChainAccepted, res)
	assert.Equal(stm.bestBlock.blk.Cid(), block1.Cid())
	assert.True(stm.KnownGoodBlocks.Has(block1.Cid()))

	// progress to block2 block on our chain
	res, err = stm.ProcessNewBlock(ctx, block2)
	assert.NoError(err)
	assert.Equal(ChainAccepted, res)
	assert.Equal(stm.bestBlock.blk.Cid(), block2.Cid())
	assert.True(stm.KnownGoodBlocks.Has(block2.Cid()))

	// Now, introduce a valid fork
	_, err = cs.Put(ctx, fork1)
	// TODO: when checking blocks, we should probably hold onto them for a
	// period of time. For now we can be okay dropping them, but later this
	// will be important.
	assert.NoError(err)

	_, err = cs.Put(ctx, fork2)
	assert.NoError(err)

	res, err = stm.ProcessNewBlock(ctx, fork3)
	assert.NoError(err)
	assert.Equal(ChainAccepted, res)
	assert.Equal(stm.bestBlock.blk.Cid(), fork3.Cid())
}

func TestRejectShorterChain(t *testing.T) {
	assert := assert.New(t)
	ctx := context.Background()
	cs := hamt.NewCborStore()
	stm := NewChainManager(cs)

	assert.NoError(stm.SetBestBlock(ctx, testGenesis))

	res, err := stm.ProcessNewBlock(ctx, block1)
	assert.NoError(err)
	assert.Equal(ChainAccepted, res)
	assert.Equal(stm.bestBlock.blk.Cid(), block1.Cid())

	res, err = stm.ProcessNewBlock(ctx, block2)
	assert.NoError(err)
	assert.Equal(ChainAccepted, res)
	assert.Equal(stm.bestBlock.blk.Cid(), block2.Cid())

	// block with lower height than our current shouldnt fail, but it shouldnt be accepted as the best block
	res, err = stm.ProcessNewBlock(ctx, fork1)
	assert.NoError(err)
	assert.Equal(ChainValid, res)
	assert.Equal(stm.bestBlock.blk.Cid(), block2.Cid())

	// block with same height as our current should fail
	res, err = stm.ProcessNewBlock(ctx, fork2)
	assert.NoError(err)
	assert.Equal(ChainValid, res)
	assert.Equal(stm.bestBlock.blk.Cid(), block2.Cid())
}