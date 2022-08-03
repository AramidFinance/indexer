# RFC Template

- Contribution Name: Conduit Framework
- Implementation Owner: Eric Warehime
- RFC PR: [algorand/indexer#????](https://github.com/algorand/indexer/pull/????)

## Summary

[summary]: #summary

The Conduit framework provides a central, service-oriented model for constructing, initializing, and running plugin-based data pipelines for Algorand.
For references regarding plugin types and their design, see the documents on
[exporters](0001-exporter-interface.md), [processors](0002-processor-interface.md), and [importers](0003-importer-interface.md).

This document describes specific implementation details around 
* Centralized configuration data
* Managing block round during normal operation, node failure, data corruption, and other failure modes that require retries, state catchup, and reboots


## Problem Statement

[problem-statement]: #problem-statement

Conduit plugins provide a modular way of defining behaviour for each of the stages of the Algorand block data pipeline, but we have no way of incorporating them into Indexer.

The existing Indexer implementation has historically incorporated new features via CLI flags, and users have always been able
to modify config variables via the CLI or via a config file. Conduit plugins present a problem to this model--dynamically
generating CLI options for plugins does not provide a positive user experience.

Indexer's model for incorporating new functionality is similarly problematic. It provides useful interfaces for various DB implementations, but 
makes assumptions about DB state, intermediate state such as the local ledger, and completeness of data being pushed through the pipeline.
This once again makes incorporating new features difficult because each addition needs to be carefully integrated into the existing pipeline.

Conduit will abstract details away from plugins allowing user to more easily develop and deploy modules which encode their specific data use cases.

## Design proposal

[design-proposal]: #design-proposal

### Conduit Pipeline Construction/Configuration


```go
package conduit

import "github.com/sirupsen/logrus"

type Pipeline interface {
	Start()
	Stop()
	
}

type PipelineImpl struct {
	logger logrus.Logger
}
```

Here is a sample config file using a 3 stage pipeline that mimics the existing Indexer--`algod` importer, local ledger processor, and postgresql exporter.
```yml
Importer:
  Name: "algod"
  Config:
    NetAddr: "127.0.0.1:8080"
    Token: "e36c01fc77e490f23e61899c0c22c6390d0fff1443af2c95d056dc5ce4e61302"
Processors:
  - Name: "block_evaluator/localledger?"
    Config:
      Catchpoint: "7560000#3OUX3TLXZNOK6YJXGETKRRV2MHMILF5CCIVZUOJCT6SLY5H2WWTQ"
      GenesisFilePath: "${HOME}/.algorand/genesis.json"
Exporter:
  Name: "postgresql"
  Config:
    ConnectionString: ""
```

### Managing Block Rounds
Conduit pipelines are concerned with processing ordered sequences of blocks from the Algorand Layer 1 network. Different plugins
within a pipeline may all separately track which blocks they have processed, or which block they expect to see next. It is the job
of the Conduit framework to provide and manage utilities for ensuring that a pipeline can operate successfully across as many
operational situations and failure modes as possible.

Conduit relies on its Exporter plugin to provide the round number during startup. Stateful exporters such as database exporters
should durably store this data and provide Conduit with the next expect round during startup.

Importer plugins are supplied with this expected block data in order to fetch new blocks to send through Conduit pipelines.
When a block fails to make if fully through the pipeline it is possible for the Conduit framework to request the same block.
For this reason we recommend that Importer implementations do not cache round numbers and act as stateless plugins--supplying exactly
what is requested by the framework.

Processor plugins can potentially pose problems to this model. Specifically, stateful processor plugins such as the Local Ledger
which stores state changes between rounds, and needs to be exactly synchronized with the exporter. To ensure that processors
cannot advance beyond the exporter round during normal operation, Conduit provides the `OnCompletion` hook to processors which will
be invoked only on successful processing by the configured Exporter. When `OnCompletion` returns an error, the framework will
redrive the given block all the way from Importer. For this reason, Exporters need to implement their Process functions idempotently.

In cases where a Processor plugin is not synchronized with the Exporter's expected round on startup, the given plugin should use
its `Init` function in order to attempt to synchronize state.

### End State

The Conduit framework will move all central logic into a Service model. Indexer currently uses a combination of 
[fetcher](https://github.com/algorand/indexer/blob/develop/fetcher/fetcher.go#L21), [importer](https://github.com/algorand/indexer/blob/develop/importer/importer.go#L12),
[indexerDb](https://github.com/algorand/indexer/blob/develop/idb/idb.go#L159), [processor](https://github.com/algorand/indexer/blob/develop/processor/processor.go#L9),
and [block handler func](https://github.com/algorand/indexer/blob/develop/cmd/algorand-indexer/daemon.go#L459) to orchestrate
the combined pipeline implementation.

Conduit will abstract some of these into their plugin equivalents, and will adapt the current goroutine fetcher-style run loops
into a single management plane that handles pipeline construction, initialization, retries, and shutdown.

### Unresolved questions

[unresolved-questions]: #unresolved-questions/FAQs
