package cmds

import (
	"context"

	"github.com/ProtoconNet/mitum2/launch"
	"github.com/ProtoconNet/mitum2/util"
	"github.com/ProtoconNet/mitum2/util/logging"
	"github.com/ProtoconNet/mitum2/util/ps"
	"github.com/rs/zerolog"
)

type CleanCommand struct { //nolint:govet //...
	launch.DesignFlag
	Vault           string `name:"vault" help:"privatekey path of vault"`
	log             *zerolog.Logger
	launch.DevFlags `embed:"" prefix:"dev."`
}

func (cmd *CleanCommand) Run(pctx context.Context) error {
	var log *logging.Logging
	if err := util.LoadFromContextOK(pctx, launch.LoggingContextKey, &log); err != nil {
		return err
	}

	log.Log().Debug().
		Interface("design", cmd.DesignFlag).
		Interface("vault", cmd.Vault).
		Interface("dev", cmd.DevFlags).
		Msg("flags")

	cmd.log = log.Log()

	pps := ps.NewPS("cmd-clean")
	_ = pps.SetLogging(log)

	_ = pps.
		AddOK(launch.PNameEncoder, launch.PEncoder, nil).
		AddOK(launch.PNameDesign, launch.PLoadDesign, nil, launch.PNameEncoder).
		AddOK(launch.PNameLocal, launch.PLocal, nil, launch.PNameDesign).
		AddOK(launch.PNameStorage, launch.PStorage, launch.PCloseStorage, launch.PNameLocal)

	_ = pps.POK(launch.PNameEncoder).
		PostAddOK(launch.PNameAddHinters, launch.PAddHinters)

	_ = pps.POK(launch.PNameDesign).
		PostAddOK(launch.PNameCheckDesign, launch.PCheckDesign)

	_ = pps.POK(launch.PNameStorage).
		PreAddOK(launch.PNameCleanStorage, launch.PCleanStorage)

	nctx := context.WithValue(pctx, launch.DesignFlagContextKey, cmd.DesignFlag)
	nctx = context.WithValue(nctx, launch.DevFlagsContextKey, cmd.DevFlags)
	nctx = context.WithValue(nctx, launch.VaultContextKey, cmd.Vault)

	cmd.log.Debug().Interface("process", pps.Verbose()).Msg("process ready")

	nctx, err := pps.Run(nctx)
	defer func() {
		cmd.log.Debug().Interface("process", pps.Verbose()).Msg("process will be closed")

		if _, err = pps.Close(nctx); err != nil {
			cmd.log.Error().Err(err).Msg("failed to close")
		}
	}()

	return err
}
