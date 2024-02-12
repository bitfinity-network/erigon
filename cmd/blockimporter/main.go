package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"runtime/pprof"
	"syscall"
	"time"

	"github.com/ledgerwatch/erigon/turbo/logging"
)

const MainNetRootKey = "308182301d060d2b0601040182dc7c0503010201060c2b0601040182dc7c05030201036100814c0e6ec71fab583b08bd81373c255c3c371b2e84863c98a4f1e08b74235d14fb5d9c0cd546d9685f913a0c0b2cc5341583bf4b4392e467db96d65b9bb4cb717112f8472e0d5a4d14505ffd7484b01291091c5f87b98883463f98091a0baaae"

func main() {
	// Parse commandline arguments
	var (
		dbPath                  = flag.String("db", "./db", "database path")
		evmUrl                  = flag.String("evm", "http://127.0.0.1:8545", "EVM canister HTTP endpoint URL")
		secondaryBlockSourceUrl = flag.String("secondary-blocks-url", "", "URL of the secondary blocks source")
		cpuprofile              = flag.String("cpuprofile", "", "write cpu profile to file")
		saveHistory             = flag.Bool("save-history-data", false, "save history data to the database")
		certifiedBlockValidator = flag.String("certificate-verification-tool", "./ic-certificate-verification-tool", "path to certified block validator executable. If non-empty will synchronize only up to latest certified block")
		evmPrincipal            = flag.String("evm-principal", "", "principal of the evm canister. Needed for certification check")
		icRootKey               = flag.String("ic-root-key", MainNetRootKey, "public key of the IC network. The default value corresponds to the mainnet.")
	)
	flag.Parse()

	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	logger := logging.GetLogger("blockimporter")

	blockCheckerSettings := BlockCheckerSettings{
		EvmPrincipal:           *evmPrincipal,
		CertificateCheckerPath: *certifiedBlockValidator,
		RootKey:                *icRootKey,
	}
	settings := Settings{
		DBPath:               *dbPath,
		Logger:               logger,
		Terminated:           make(chan struct{}),
		RetryCount:           100,
		RetryInterval:        time.Second,
		PollInterval:         time.Second,
		SaveHistoryData:      *saveHistory,
		blockCheckerSettings: blockCheckerSettings,
	}

	c := make(chan os.Signal, 10)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		close(settings.Terminated)
	}()

	blockSource := NewHttpBlockSource(*evmUrl)
	var secondaryBlockSource BlockSource
	if *secondaryBlockSourceUrl != "" {
		secondarySource := NewHttpBlockSource(*secondaryBlockSourceUrl)
		secondaryBlockSource = &secondarySource
	}

	err := RunImport(&settings, &blockSource, secondaryBlockSource)

	if err != nil {
		logger.Error(err.Error())
		panic(err)
	}
}
