package cmds

import (
	"context"
	"os"
	"strings"

	"github.com/ProtoconNet/mitum2/util"
	"github.com/ProtoconNet/mitum2/util/valuehash"
	"github.com/pkg/errors"
)

type NetworkClientStateCommand struct { //nolint:govet //...
	BaseNetworkClientCommand
	Key  string `arg:"" name:"state key" help:"state key"`
	Hash string `arg:"" name:"state hash" help:"state hash" default:""`
}

func (cmd *NetworkClientStateCommand) Run(pctx context.Context) error {
	if err := cmd.Prepare(pctx); err != nil {
		return err
	}

	defer func() {
		_ = cmd.Client.Close()
	}()

	if len(strings.TrimSpace(cmd.Key)) < 1 {
		return errors.Errorf("empty state key")
	}

	var h util.Hash

	if len(strings.TrimSpace(cmd.Hash)) > 0 {
		h = valuehash.NewBytesFromString(cmd.Hash)
	}

	ctx, cancel := context.WithTimeout(pctx, cmd.Timeout)
	defer cancel()

	switch st, found, err := cmd.Client.State(ctx, cmd.Remote.ConnInfo(), cmd.Key, h); {
	case err != nil:
		cmd.Log.Error().Err(err).Msg("failed to get state")

		return err
	case !found:
		cmd.Log.Error().Msg("not found")

		return nil
	case h != nil && st == nil:
		cmd.Log.Info().Msg("no new state")

		return nil
	default:
		return cmd.Print(st, os.Stdout)
	}
}
