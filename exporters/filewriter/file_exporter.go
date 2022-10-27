package filewriter

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"

	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"

	"github.com/algorand/indexer/data"
	"github.com/algorand/indexer/exporters"
	"github.com/algorand/indexer/plugins"
	"github.com/algorand/indexer/util"
)

const (
	exporterName = "file_writer"
	// FilePattern is used to name the output files.
	FilePattern = "%[1]d_block.json"
)

type fileExporter struct {
	round  uint64
	cfg    Config
	logger *logrus.Logger
}

var fileExporterMetadata = exporters.ExporterMetadata{
	ExpName:        exporterName,
	ExpDescription: "Exporter for writing data to a file.",
	ExpDeprecated:  false,
}

// Constructor is the ExporterConstructor implementation for the filewriter exporter
type Constructor struct{}

// New initializes a fileExporter
func (c *Constructor) New() exporters.Exporter {
	return &fileExporter{}
}

func (exp *fileExporter) Metadata() exporters.ExporterMetadata {
	return fileExporterMetadata
}

func (exp *fileExporter) Init(_ context.Context, initProvider data.InitProvider, cfg plugins.PluginConfig, logger *logrus.Logger) error {
	exp.logger = logger
	var err error
	exp.cfg, err = unmarshalConfig(string(cfg))
	if err != nil {
		return fmt.Errorf("connect failure in unmarshalConfig: %w", err)
	}
	if exp.cfg.FilenamePattern == "" {
		exp.cfg.FilenamePattern = FilePattern
	}
	// create block directory
	err = os.Mkdir(exp.cfg.BlocksDir, 0755)
	if err != nil && errors.Is(err, os.ErrExist) {
		// Ignore mkdir if the dir exists
		err = nil
	} else if err != nil {
		return fmt.Errorf("Init() error: %w", err)
	}
	exp.round = uint64(initProvider.NextDBRound())
	return err
}

func (exp *fileExporter) Config() plugins.PluginConfig {
	ret, _ := yaml.Marshal(exp.cfg)
	return plugins.PluginConfig(ret)
}

func (exp *fileExporter) Close() error {
	exp.logger.Infof("latest round on file: %d", exp.round)
	return nil
}

func (exp *fileExporter) Receive(exportData data.BlockData) error {
	if exp.logger == nil {
		return fmt.Errorf("exporter not initialized")
	}
	if exportData.Round() != exp.round {
		return fmt.Errorf("Receive(): wrong block: received round %d, expected round %d", exportData.Round(), exp.round)
	}

	// write block to file
	{
		if exp.cfg.DropCertificate {
			exportData.Certificate = nil
		}

		blockFile := path.Join(exp.cfg.BlocksDir, fmt.Sprintf(exp.cfg.FilenamePattern, exportData.Round()))
		err := util.EncodeToFile(blockFile, exportData, true)
		if err != nil {
			return fmt.Errorf("Receive(): failed to write file %s: %w", blockFile, err)
		}
		exp.logger.Infof("Wrote block %d to %s", exportData.Round(), blockFile)
	}

	exp.round++
	return nil
}

func unmarshalConfig(cfg string) (Config, error) {
	var config Config
	err := yaml.Unmarshal([]byte(cfg), &config)
	return config, err
}

func init() {
	exporters.RegisterExporter(exporterName, &Constructor{})
}