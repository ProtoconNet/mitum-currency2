package cmds

type CurrencyCommand struct {
	CreateAccount         CreateAccountCommand         `cmd:"" name:"create-account" help:"create new account"`
	UpdateKey             UpdateKeyCommand             `cmd:"" name:"update-key" help:"update account keys"`
	Transfer              TransferCommand              `cmd:"" name:"transfer" help:"transfer"`
	RegisterCurrency      RegisterCurrencyCommand      `cmd:"" name:"register-currency" help:"register new currency"`
	UpdateCurrency        UpdateCurrencyCommand        `cmd:"" name:"update-currency" help:"update currency policy"`
	CreateContractAccount CreateContractAccountCommand `cmd:"" name:"create-contract-account" help:"create new contract account"`
	UpdateHandler         UpdateHandlerCommand         `cmd:"" name:"update-handler" help:"update handler of contract account"`
	Withdraw              WithdrawCommand              `cmd:"" name:"withdraw" help:"withdraw amounts from target contract account"`
}
