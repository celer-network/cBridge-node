# cBridge Relay Node

Official implementation of cBridge relay node in Golang.

## Prerequisites

### Prepare Machine

To run a cBridge relay node, it is recommended to prepare a Linux machine with a minimum of 4GB RAM, at least 2 cores and at least 20GB of disk space.

### Install and Run CockroachDB

cBridge relay node uses CockroachDB to save transfer records. Please follow the below steps to install and setup CockroachDB on your machine.

1. Install CockroachDB by following the [official guide](https://www.cockroachlabs.com/docs/v20.2/install-cockroachdb-linux).

2. Start CockroachDB

```sh
cockroach start-single-node --insecure --listen-addr=localhost:26257 --store=path=/tmp/relaynode_sql_db
```

Note that `26257` is the RPC port. You can change it to any port you prefer but please note it down as you will need it  later when configuring the relay node.

3. Under the root of this repo, execute the following command to initialize the DB table for the relay node:

```sh
cockroach sql --insecure --host=127.0.0.1 --port=26257 --database=cbridge < ./server/schema.sql
```

## Get cBridge Node Binanry

### Download the Binary

You can directly download and unzip the cBridge node binary:

- MacOS

```sh
wget -c https://get.celer.app/cbridgenode/cbridge-node-v1.0-darwin-amd64.tar.gz -O - | tar -xz 
```

MD5 Checksum for the unzipped binary: `7c6c7601b0a590e26b3dc3e633a558bb`

- Linux AMD64

```sh
wget -c https://get.celer.app/cbridgenode/cbridge-node-v1.0-linux-amd64.tar.gz -O - | tar -xz 
```

MD5 Checksum for the unzipped binary: `8f8d166de42a21e7c2cd27554f0da129`

- Linux ARM64

```sh
wget -c https://get.celer.app/cbridgenode/cbridge-node-v1.0-linux-arm64.tar.gz -O - | tar -xz 
```

MD5 Checksum for the unzipped binary: `57bc107e0ac6814bbef69409a887c972`

### Build the Source

You can also get the binary by building from the sources. Please first make sure you have installed Golang (version 1.15+). Then under the root of this repo, execute

```sh
go build -o cbridge-node ./server/main/
```

## Configure Your Relay Node

(**IMPORTANT**) Before we start to configure the relay node, please make sure that the following things have been done:

1. CockroachDB is up running, and all the database and tables has been initialized (as instructed above).
2. Prepare your account key store JSON file and key store password.
3. Make sure the account has enough funds for the chains and the tokens you would like the relay node to bridge. Please refer [here](https://cbridge-stat.s3.us-west-2.amazonaws.com/mainnet/chains-tokens.json) for the list of chains and tokens currently supported in cBridge. Note that there is a minimum balance requirement for each chain and token. For example, if you want to bridge the USDC token from other chains to Ethereum Mainnet, you need to make sure that your account has at least 10,000 USDC balance. In general, it is recommended to have a much larger balance in your wallet than the minimum requirement to improve your chance of being selected for transfers and thus increase your collected transaction fees.
4. Make sure the account has enough gas tokens (e.g., ETH for Ethereum Mainnet, BNB for BSC) to cover the gas cost incurred during relay node operations. Note that although the gas fee has already been included in the fee charged to the user, it is paid in the transferred token (instead of the gas token), and thus reserving some gas tokens is still necessary.

If you have checked all of the above items, let's proceed to the relay node config file. An example of the config file is enclosed under `./env`:

```javascript
{
  "chainConfig": [
    // Specify the list of chains + tokens the relay node support
    // Your node will only relay transfers where both the source and the destination chain
    // are within the below chain list.
    // For example, if the list contains Ethereum Mainnet (chainId=1), BSC (chainId=56)
    // and Arbitrum (chainId=42161), the node can relay (1) Ethereum Mainnet --> BSC
    // (2) BSC --> Ethereum Mainnet (3) Arbitrum --> Ethereum Mainnet
    // (4) Ethereum Mainnet --> Arbitrum (5) Arbitrum --> BSC (6) BSC --> Arbitrum
    {
      "chainId": 1, //Ethereum Mainnet
      "endpoint": "", // specify your chain RPC url (e.g., an Infura endpoint)
      "contractAddress": "0x841ce48F9446C8E281D3F1444cB859b4A6D0738C", // cBridge contract address
      "feeRate": 5, // fee percent the node charge for transfers destined to this chain (in unit of 0.01%, 5 means 0.05%)
      "tokenConfig": [
        // Specify the list of tokens the relay node supports on the chain
        // NOTE: if you specify a token on the chain, the minimum balance requirement should be met
        // otherwise the relay node may fail to operate
        // Refer to https://cbridge-stat.s3.us-west-2.amazonaws.com/mainnet/chains-tokens.json 
        // about the minimum balance requirement for each chain and token
        {
          "tokenName": "USDT",
          "tokenAddress": "0xdac17f958d2ee523a2206206994597c13d831ec7",
          "tokenDecimal": 6
        },
        {
          "tokenName": "DAI",
          "tokenAddress": "0x6b175474e89094c44da98b954eedeac495271d0f",
          "tokenDecimal": 18
        },
        {
          "tokenName": "USDC",
          "tokenAddress": "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
          "tokenDecimal": 6
        },
        {
          "tokenName": "BUSD",
          "tokenAddress": "0x4fabb145d64652a948d72533023f6e7a623c7c53",
          "tokenDecimal": 18
        }
      ],
      "watchConfig": {
        "pollingInterval": 10, // interval (in seconds) to periodically query eth logs
        "blockDelay": 4, // the block confirmation delay (in blocks)
        "maxBlockDelta": 5000 // maximum number of blocks for each eth log query
      }
    },
    {
      "chainId": 56, // Binance Smart Chain
      "endpoint": "",
      "contractAddress": "0x841ce48F9446C8E281D3F1444cB859b4A6D0738C",
      "feeRate": 0,
      "tokenConfig": [
        {
          "tokenName": "USDT",
          "tokenAddress": "0x55d398326f99059ff775485246999027b3197955",
          "tokenDecimal": 18
        },
        {
          "tokenName": "DAI",
          "tokenAddress": "0x1af3f329e8be154074d8769d1ffa4ee058b1dbc3",
          "tokenDecimal": 18
        },
        {
          "tokenName": "USDC",
          "tokenAddress": "0x8ac76a51cc950d9822d68b83fe1ad97b32cd580d",
          "tokenDecimal": 18
        },
        {
          "tokenName": "BUSD",
          "tokenAddress": "0xe9e7cea3dedca5984780bafc599bd69add087d56",
          "tokenDecimal": 18
        }
      ],
      "watchConfig": {
        "pollingInterval": 3,
        "blockDelay": 4,
        "maxBlockDelta": 5000
      }
    },
    {
      "chainId": 42161, // Arbitrum
      "endpoint": "",
      "contractAddress": "0x841ce48F9446C8E281D3F1444cB859b4A6D0738C",
      "feeRate": 0,
      "tokenConfig": [
        {
          "tokenName": "USDC",
          "tokenAddress": "0xFF970A61A04b1cA14834A43f5dE4533eBDDB5CC8",
          "tokenDecimal": 6
        }
      ],
      "watchConfig": {
        "pollingInterval": 3,
        "blockDelay": 4,
        "maxBlockDelta": 5000
      }
    }
  ],
  "ksPath": "", // relay node keystore json file path
  "ksPwd": "", // keystore password (never commit it to public repo)
  "db": "127.0.0.1:26257", // CockroachDB RPC url
  "gateway": "cbridge-api.celer.network:8081" //cBridge gateway server url
}
```

## Start Your Relay Node

Suppose that the config file is saved to `./env/config.json`. Then you can start the relay node by executing the binary:

```sh
./cbridge-node -p 8088 -c ./env/config.json
```

where `-c` specifies the path to the node config file and `-p` specifies the http port that you can use to query tx and stats about your relay node (to be introduced later).

After a while, you should see your relay node is up running.

**NOTE**: When the relay node is started for the first time, it needs to Approve the allowance for each ERC-20 token you specified on each chain, which might might take a few minutes. In addition, please make sure that there are enough gas tokens to cover the Approve tx gas fee, otherwise the node may not be successfully started (it will show the error msg `Error when approving token DAI on chain 3: failed to estimate gas needed: insufficient funds for transfer`).

## Query Relay Node Stats

While the relay node is running, you can query the node stats by

```sh
curl http://localhost:8088/v1/summary/total
```

Remember to change `8088` to the port you specify when starting the node. Then you should see the stats about your node. For example,

```
------------------------------------------------
chain 1 -> chain 56
Received 256 transfers
Successfully processed 234 transfers
Success rate: 91.40%
Token name: USDC, transfer volume:25000 USDC, earned fee:25.31 USDC
Token name: DAI, transfer volume:9800 USDC, earned fee:12.58 DAI
------------------------------------------------
chain 42161 -> chain 1
Received 1678 transfers
Successfully processed 1620 transfers
Success rate: 96.54%
Token name: USDC, transfer volume:96250 USDC, earned fee:150.96 USDC
------------------------------------------------
```

You can also query the detailed transactions by

```sh
curl http://localhost:8088/v1/transfer/100
```

This will show the latest 100 transaction entries.

## cBridge Network Stats

Please check out this [JSON file](https://cbridge-stat.s3.us-west-2.amazonaws.com/mainnet/cbridge-stat.json) where we periodically update the global statistics about cBridge network, such as the tx volume and the global relay node info.

**Notes on cBridge Network Stats JSON**

- `totalTx` is the number of transfers processed so far; `last24HourTx` is the number of transfers processed in the last 24 hours
- `totalTxVolume` is the total amount of transfers processed so far (in USD); `last24HourTxVolume` is the total amount of transfers processed in the last 24 hours (in USD).
- `relayNodes` shows all the nodes that have accepted at least 1 transfers in the last 24 hours. The number of nodes in the list may be slightly different from `activeRelayNodeNum` which only accounts for the currently online nodes.
- `fee` is a float number that corresponds to the portion of each transfer amount collected as the fee (e.g., 0.001 means the node charges 0.001*transfer_amount tx fee on top of the gas fee part).

## Best Practices for Operating A Relay Node

To make sure your relay node can provide a better service for users and increase your profits, there are some best practices to follow.

### Liveness Monitoring and DevOps

A relay node may unexpectedly die for some reasons (e.g., the machine is down, or the relay node process is unexpectedly killed by the system). In order to improve your relay node uptime, it is highly recommended to take the following actions items:

- Add a monitoring for the machine liveness
- Add a monitoring for the relay node process liveness on the machine
- Add some basic DevOps tools so that the relay node can automatically recover (e.g., systemd)

### Liquidity Monitoring and Rebalancing

When your relay node runs out of liquidity tokens, it won't be able to accept new transfer requests. It is recommended to add monitoring for your relay node's balance on each chain when it drops below a threshold. When necessary, you may rebalance the liquidity among different chains.

### Gas Token Balance Monitoring

The relay node may also run of the gas tokens and get stuck on when sending transactions. It is also recommended to add monitoring for the gas tokens on each chain when their balances drop below a threshold.

### Fee Adjustment

In cBridge, relay nodes with a lower transaction fee and a higher success rate have a higher probability of being selected to serve cross-chain transfer requests. As a result, besides improving the service stability of your relay node, it is also important to promptly adjust your fee rate to be competitive in the market. You can query the latest cBridge network stat [here](https://cbridge-stat.s3.us-west-2.amazonaws.com/mainnet/cbridge-stat.json) where you can view the transaction stats for all the relay nodes and their fee schedules.

Moreover, you may consider using a lower fee rate (even 0%) for transfers in some directions in order to maintain balanced transfers. A typical example is between L2 Arbitrum and L1 Ethereum Mainnet. The fee for L2-to-L1 may be higher due to the saving of 7-day challenge period for users while the fee for L1-to-L2 should be lower (even zero) to incentivize users to also use your node for the L1-to-L2 transfers. Appropriate balancing can help avoid manual rebalancing your liquidity among L1 and L2.

### Blockchain Infrastrucutre

cBridge relay node relies on a stable blockchain infrasture (e.g., Infura) for on-chain event monitoring and function calls. Please make sure that relay node won't exceed your daily quota limit.

Here is a simple way to calculate your daily requests to the blockchain infra:

- Suppose your configured pollingInterval = X seconds. Then the number of requests per day per chain is 4320000/X. For example, if X=10, then the daily requests for that chain will be 43200 and your daily request limit should be more than this value.

**NOTE**: If you use Infura, it is recommended to use a paid plan to ensure your service quality and increase your success rate.

### Relay Node Security

Since the relay node funds never leave its wallet, it is your responsibility to make sure the private keys of your relay node is safely stored (e.g., never commit your keystore password to any public repo or public machine).

Moreover, please properly restrict the access to your open ports on the relay node machine. (e.g., the port for CockroachDB and the port for quering relay node transaction stats)

## Contracts

The multi-hop transfer via relay node is secured by the underlying cBridge smart contract. Please check the contract source code in <https://github.com/celer-network/cBridge-contracts>

## Relay Node FAQ

### How is the fee calulated for each transfer?

For each transfer, the fee is calculated as

```
fee = relay_node_gas_cost + fee_percentage * transfer_amount,
```

where the first part is used to cover the gas cost incurred to the relay node and the second part is additional transaction fee collected by the relay node as economic incentives.

### How is the relay node selected for each transfer?

In cBridge v1.0, the relay node for each transfer request is auto-selected by our gateway server to provide better user experience. The selection criteria is based on the historical relay success rate,  the charged transaction fee rate and the number of relayed transfers. In general, the higher the success rate, the lower the transaction fee rate and the smaller number of relayed transfers, the higher chance of being selected.

In the future releases, we will also support the manual selection mode where the user can see the list of online relay nodes and manually select the preferred one.

### Where are the relay node funds entrusted?

In cBridge, the relay node’s funds are fully non-custodian, which means that the relay node operator has the full control of the funds.

### Who can operate a cBridge relay node? Is whitelist required?

cBridge node is open-sourced to the community. Anyone with a machine and liquidity can become a relay node. There is no whitelist.

### What is the benefit of running a relay node?

The relay node can earn a transaction fee for each transaction it successfully relayed.

### Would the relay node lose funds if the user is misbehaved?

No, the relay node funds are fully secured with the underlying Hashed Time Lock (HTL) contract even if the user could be malicious, and in our implementation the relay node will respond properly in different scenarios where the user is misbehaved

- If the user does not release the fund to the relay node, it will automatically cancel its HTL transfer on the destination chain after the HTL deadline has passed.
- If the user directly claims the funds on the destination chain without revealing the secret on the source chain, the relay node will automatically claims the funds on the source chain.

### What is the behavior after the relay node recovers from a failure?

If the relay node dies, it won't accept new transfer requests from users. In the case where there are in-progress transfers when the relay node dies, it will automatically pick up these transfers after it restarts.
