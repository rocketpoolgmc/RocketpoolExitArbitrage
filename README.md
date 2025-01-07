# Rocket Pool Distribute Arbitrage CLI Tool

> **WARNING**  
> This software is an **initial version** and **has not been thoroughly tested**.  
> Use **at your own risk**. We do **not** provide any guarantees or assume any liability for any potential losses or damages.  
> Always review the code and run your own tests before using it in a production environment.

---

## Overview

This repository contains a CLI (Command-Line Interface) tool designed to help **capture arbitrage opportunities** created by distributing minipools.  
Whenever you call **distribute** on a pool, it sends the Rocket Pool share to the rETH contract, allowing more rETH to be burned in exchange for ETH.  
When **exiting minipools** or **claiming ETH**, it is vital to check for arbitrage opportunities that would otherwise be captured by third parties.


The core objective is to leverage **distribute** calls in combination with **rETH burn** to capture potential arbitrage gains. This tool can also facilitate a **flashloan** for users who don't already hold rETH but want to capitalize on the arbitrage.

### Why This Tool?

- **Scenario 1**:  
  If you already hold rETH, you can bundle the **distribute** action with an **rETH burn** to recive ETH at the protocol rate.
- **Scenario 2**:  
  If you don’t hold rETH, you can use a **flashloan** to obtain rETH, perform the distribute and burn, then repay the loan—collecting any remaining ETH as profit.

---

## Table of Contents

1. [Requirements](#requirements)  
2. [Installation](#installation)  
3. [Known Issues & Limitations](#known-issues--limitations)  
4. [Smart Contract](#smart-contract)
5. [Usage](#usage)  
   - [Scenario 1: If You Already Have rETH](#scenario-1-if-you-already-have-reth)  
   - [Scenario 2: Flashloan Approach](#scenario-2-flashloan-approach)  
6. [Configuration](#configuration)  
7. [Stretch Goals / Future Enhancements](#stretch-goals--future-enhancements)  
8. [License](#license)

---

## Requirements

1. **Go (version 1.22+ recommended)**
   - You can download and install Go from the official [Go Downloads page](https://go.dev/dl/).  
   - Refer to the official [Getting Started](https://go.dev/doc/install) guide for further instructions.  
   - Earlier versions (e.g., 1.20, 1.21) may still work, but are **not** tested.

2. **Access to a Web3 Provider**
   - Typically this is your **Rocket Pool Eth1 client** (e.g., Geth, Nethermind).  
   - Ensure that your Rocket Pool setup has `Expose RPC Ports` configured to `Open to Localhost`.

3. **Minipool Exit Completed**
   - To finalize a minipool and distribute the full 32 ETH, the validator must be **exited from the consensus layer**.  
   - Use the **Rocket Pool** command `rocketpool m e` to initiate the exit.  
   - Wait until the exit is fully processed and ETH is withdrawn from the consensus layer before proceeding with distribution.  
   - You can monitor progress in your node logs or by using on-chain explorers to confirm that ETH has been returned.

---

## Installation

For this initial version, **no binaries are provided**. You will need to **install Go** (1.22+ recommended) and build from source:

1. **Clone this repository**:

    ```bash
    git clone https://github.com//0xtrooper/RocketpoolExitArbitrage.git
    cd RocketpoolExitArbitrage
    ```

2. **Build the binary**:

    ```bash
    go build ./cmd/distribute/
    ```

3. **Run the CLI tool**:

    ```bash
    ./distribute --help
    ``` 

--

## Known Issues & Limitations

- **Inefficient Large Arbitrage Swaps**  
  Currently, the tool only integrates with Uniswap, making large arbitrage swaps inefficient. Future updates with aggregators (probably 1inch) will improve performance for bigger transactions.

- **Profit Checks with Multiple Pools**  
  When operating with low discounts and a large number of pools using the `--minipools` flag, the tool does not perform individual profit checks for each pool. This can lead to suboptimal profits across multiple pools. It is recommended to limit the number of pools in a single call to ensure better profitability at times of low discount. **This will improved in a further update**

- **Lack of Comprehensive Tests**  
  This initial version lacks thorough testing, especially for the local rETH functionality. Testing for the local rETH integration is planned and will be completed in the coming days to enhance reliability and stability.

--

## Smart Contract

To execute a Flashswap with Uniswap and simultaneously burn rETH within a single transaction, we developed and deployed a custom smart contract. This contract is fully verified on Etherscan and can be viewed [here](https://etherscan.io/address/0xdb7DA96A75B43927Da9C0321cD2F82c430305CA0#code). 

### Key Features
- **No Approvals Required:** The contract operates without needing any external approvals, simplifying its usage and reducing potential points of failure.
- **ETH-Free Transactions:** The transaction process does not involve sending any ETH, minimizing exposure to ETH-related risks.

### Security Considerations
While the smart contract is designed with streamlined functionality and minimal interaction with external elements, it presents a low-risk profile. However, it is important to note that this contract has not undergone a formal security audit. Users are encouraged to review the contract code on Etherscan and exercise caution when interacting with it. Always act based on your best knowledge and understanding, ensuring you are comfortable with the contract's operations and potential risks before proceeding.


--

## Configuration

This CLI tool is configured primarily through command-line flags. Below is a list of the flags and their functions:


---

## Local rETH

- **Flag**: `--local-reth`  
  **Type**: boolean  
  **Default**: `false`  
  **Description**: Use existing local rETH instead of taking a flashloan. If `false`, the CLI attempts a flashloan. Use this if you want to convert rETH to ETH at the primary ratio.
  **Example**:
  ```bash
  ./distribute --local-reth
  ```

---

## Minipool (Single)

- **Flag**: `--minipool`  
  **Type**: string  
  **Default**: (empty)  
  **Description**: Single minipool address to distribute. Use this if you only want to distribute to one minipool at a time.  
  **Example**:
  ```bash
  ./distribute --minipool="0xABC123..."
  ```

---

## Minipool (Multiple)

- **Flag**: `--minipools`  
  **Type**: string (comma-separated)  
  **Default**: (empty)  
  **Description**: Comma-separated list of minipool addresses to distribute. This does not reduce the gas fee per distribute, but only one arbitrage call is needed.
  **Example**:
  ```bash
  ./distribute --minipools="0xABC123...,0xDEF456...,0x789ABC..."
  ```

---

## Debugging

- **Flag**: `--debug`  
  **Type**: boolean  
  **Default**: `false`  
  **Description**: Enables detailed debug logs. If set to `true`, the CLI outputs verbose diagnostic information.  
  **Example**:
  ```bash
  ./distribute --debug
  ```

---

## Command Override

- **Flag**: `--command`  
  **Type**: string  
  **Default**: `docker exec rocketpool_node /go/bin/rocketpool`  
  **Description**: Overrides the default command used to run the Rocket Pool smartnode daemon. Adjust if your container or binary path differs.  
  **Example**:
  ```bash
  ./distribute --command="docker exec my_node /go/bin/rocketpool"
  ```

---

## Flashbots Gas Refund Address

- **Flag**: `--refund-address`  
  **Type**: string  
  **Default**: (empty; if not set, a random searcher address is generated)  
  **Description**: Address to receive gas refunds from Flashbots. Flashbots users are automatically eligible to receive gas fee refunds. If Flashbots can include a bundle on chain for a lower price, you are eligible to receive a refund. Those are automatically paid without one week. For more information: https://docs.flashbots.net/flashbots-auction/advanced/gas-fee-refunds
  **Example**:
  ```bash
  ./distribute --refund-address="0x12345..."
  ```

---

## Flashbots Searcher Private Key

- **Flag**: `--searcher-private-key`  
  **Type**: string  
  **Default**: (empty; if not set, a random key is generated)  
  **Description**: **Completly optional!** Private key for the searcher used in Flashbots transactions. Flashbots uses a repupation based system to controll access in times of high demand. For more information: https://docs.flashbots.net/flashbots-auction/advanced/reputation 
  **Example**:
  ```bash
  ./distribute --searcher-private-key="abcdef123456..."
  ```

---

## Ethereum RPC Endpoint

- **Flag**: `--rpc`  
  **Type**: string  
  **Default**: `http://localhost:8545`  
  **Description**: Usually, this is the Rocket Pool eth1 client. Alternatively, you can specify a different RPC endpoint if needed. Use `--rpc-port` if you only want to set a non-default port.
  **Example**:
  ```bash
  ./distribute --rpc="https://mainnet.infura.io/v3/YOUR_PROJECT_ID"
  ```

---

## RPC Port

- **Flag**: `--rpc-port`  
  **Type**: string  
  **Default**: `8545`  
  **Description**: Use this flag if you are using a non-default port for the Rocket Pool eth1 client. Alternatively, you can use a different RPC endpoint with the `--rpc` flag.
  **Example**:
  ```bash
  ./distribute --rpc-port="9545"
  ```

---

## Skip Confirmation Prompt

- **Flag**: `--skip-confirmation` / `--y`  
  **Type**: boolean  
  **Default**: `false`  
  **Description**: If set, the CLI will skip the confirmation prompt before executing, which speeds up automated runs.  
  **Examples**:
  ```bash
  # Long flag
  ./distribute --skip-confirmation

  # Short flag
  ./distribute -y
  ```

---

## Profit Checks

- **Flag**: `--check-profit`  
  **Type**: boolean  
  **Default**: `true`  
  **Description**: If set to `true`, the CLI reverts when the expected profit is too low.  
  **Example**:
  ```bash
  ./distribute --check-profit=false
  ```

- **Flag**: `--ignoreDistributeCost`  
  **Type**: boolean  
  **Default**: `false`  
  **Description**: Reverts when the profit is too low, but does **not** consider the cost of the distribute call(s). Best used if you want to distribute rewards regardless of minor profit, but check for arbitrage at the same time.
  **Example**:
  ```bash
  ./distribute --ignoreDistributeCost
  ```

---

## Dry Run Mode

- **Flag**: `--dryRun`  
  **Type**: boolean  
  **Default**: `false`  
  **Description**: Performs a dry run without sending the bundle to Flashbots; only prints the transaction bundle.  
  **Example**:
  ```bash
  ./distribute --dryRun
  ```

---

## Combining Flags

You can combine multiple flags in a single command. For example:

```bash
./distribute --local-reth --minipools="0xABC123...,0xDEF456..." --debug
```

This example enables debug logs, uses local rETH and specifies multiple minipools.

---

## Stretch Goals / Future Enhancements

Future enhancements aim to improve the tool's functionality, efficiency, and reliability:

- **Integration with Aggregators:** Currently, the tool only integrates with Uniswap, which limits the efficiency of large arbitrage swaps. Integrating with aggregators such as 1inch is planned to enhance performance for larger transactions. This integration will require an additional smart contract to utilize flash loans.

- **Individual Profit Checks for Multiple Pools:** When operating with low discounts and a high number of pools using the `--minipools` flag, individual profit checks for each pool are not performed, potentially leading to suboptimal profits. Implementing individual profit checks will ensure better profitability when managing multiple pools. As this feature is only needed for low discount situations, it is currently skipped (at the time of writing, there is a -0.4% discount).

- **Comprehensive Testing:** The initial version lacks thorough testing, particularly for the local rETH functionality. Extensive testing for the local rETH integration is scheduled to be completed in the near future. New minipools have been started on Holesky for testing, which is the main bottleneck for these tests.

- **Enhanced Arbitrage Checks:** Introducing a current arbitrage check feature will calculate the existing discount ratio and determine the potential profit from either exiting the position or distributing funds. This will provide users with better insights and decision-making capabilities regarding their arbitrage strategies.

These improvements are designed to address current limitations and provide a more robust and efficient user experience in upcoming releases.

--

## License

This project is open source and available under the [MIT License](LICENSE).

