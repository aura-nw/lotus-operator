# Lotus Operator

Secure Bridge Transaction Verification and Validation.

The Lotus operator is a critical component ensuring the integrity and security of bridge transactions between the Bitcoin and Aura networks with EVM compatibility. It plays a vital role in verifying and validating income and outcome transactions, safeguarding the smooth flow of assets across these chains.

## 1. Config

The operator service need a config file located at `./operator.toml` for default. Here is describe of fields in config:

a. Server

* http-port: This defines the port number on which the operator server listens for incoming connections.

b. Bitcoin

* `network`: Specifies the Bitcoin network to connect to (likely "testnet3" for a test network in this case).

* `host`: The IP address and port of the Bitcoin node used for communication.
* `user`: The username for authentication with the Bitcoin node (if required).
* `password`: The password for the Bitcoin node user (if required).
* `query-interval`: The interval (in seconds) at which the bridge queries the Bitcoin node for new transactions.
* `min-confirmations`: The minimum number of confirmations required for a Bitcoin transaction before it's considered for bridging.
* `bitcoin-multisig`: The multisignature address used for Bitcoin transactions on the bridge.
* `private-key`: The private key associated with the bridge's multisignature address (likely obfuscated for security reasons).
* `redeem-script`: The redeem script for the multisignature address (likely obfuscated).

c. Evm

* `url`: The URL of the Aura Network JSON RPC endpoint for communication.
* `chain-id`: The chain ID of the Aura Network used by the bridge.
* `query-interval`: The interval (in seconds) at which the bridge queries Aura Network for transaction confirmations.
* `min-confirmations`: The minimum number of confirmations required for an Aura Network transaction before it's considered finalized.
* `private-key`: The private key used by the bridge for signing transactions on Aura Network (likely obfuscated).
* `call-timeout`: The timeout value (in seconds) for making calls to the Aura Network JSON RPC endpoint.

## 2. Run

After editing config properly. Run the service using command:

```bash
make run
```
