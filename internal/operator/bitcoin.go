package operator

type BtcVerifier interface {
	VerifyBtcDeposit(txHash string, utxo string) (bool, error)
	VerifyTokenDeposit(txHash string) (bool, error)
	VerifyInscriptionDeposit(txHash string) (bool, error)
}
