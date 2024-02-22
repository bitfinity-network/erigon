package main

import (
	"encoding/json"
	"fmt"
	"os/exec"

	"github.com/ledgerwatch/erigon/core/types"
)

type BlockCheckerSettings struct {
	EvmPrincipal           string
	CertificateCheckerPath string
	RootKey                string
}

type BlockChecker interface {
	CheckBlock(block types.Block) error
}

func NewBlockChecker(settings BlockCheckerSettings, blocksSource BlockSource) BlockChecker {
	if settings.EvmPrincipal == "" || settings.CertificateCheckerPath == "" || settings.RootKey == "" {
		return dummyBlockChecker{}
	} else {
		return newToolBlockChecker(settings, blocksSource)
	}
}

type dummyBlockChecker struct{}

func (checker dummyBlockChecker) CheckBlock(block types.Block) error {
	return nil
}

type blockData struct {
	response    CertifiedBlockData
	blockHeader types.Header
}

type toolBlockChecker struct {
	settings     BlockCheckerSettings
	lastBlock    *blockData
	blocksSource BlockSource
}

func newToolBlockChecker(settings BlockCheckerSettings, blocksSource BlockSource) *toolBlockChecker {
	return &toolBlockChecker{
		lastBlock:    nil,
		settings:     settings,
		blocksSource: blocksSource,
	}
}

func (checker *toolBlockChecker) CheckBlock(block types.Block) error {
	for checker.lastBlock == nil || checker.lastBlock.blockHeader.Number.Uint64() < block.NumberU64() {
		if lastBlock, err := checker.getLastBlockData(); err != nil {
			return err
		} else {
			checker.lastBlock = &lastBlock
		}
	}

	if checker.lastBlock.blockHeader.Number.Uint64() > block.NumberU64() {
		return nil
	}

	if checker.lastBlock.blockHeader.Root != block.Root() {
		return fmt.Errorf("certified block contains different root, have: %s, want %s", checker.lastBlock.blockHeader.Root.String(), block.Root())
	}

	vertifiedResponse, err := json.Marshal(checker.lastBlock.response)
	if err != nil {
		return fmt.Errorf("failed to serialize certified response: %s", err)
	}

	_, err = exec.Command(checker.settings.CertificateCheckerPath, string(vertifiedResponse), checker.settings.EvmPrincipal, checker.settings.RootKey).Output()
	if err != nil {
		return err
	}

	checker.lastBlock = nil
	return nil
}

func (checker *toolBlockChecker) getLastBlockData() (blockData, error) {
	data, err := checker.blocksSource.GetLastCertifiedBlockData()
	if err != nil {
		return blockData{}, fmt.Errorf("failed to get last certified block: %s", err)
	}

	var header types.Header
	if err = json.Unmarshal(data.Block, &header); err != nil {
		return blockData{}, fmt.Errorf("failed to parse block header: %s", err)
	}

	return blockData{
		response:    data,
		blockHeader: header,
	}, nil
}
