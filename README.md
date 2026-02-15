# TON W5 Vanity Address Generator

A fast CLI tool for generating **TON Wallet V5 (W5R1)** vanity
addresses.

It continuously generates random mnemonics and checks whether the
resulting wallet address matches a user-defined pattern.

------------------------------------------------------------------------

## Features

-   Multi-threaded (goroutines)
-   Custom vanity pattern search
-   Optional case-sensitive matching
-   Optional live speed statistics
-   Saves matching address + mnemonic to file
-   Windows sleep prevention support

------------------------------------------------------------------------

## Installation

### Option 1 --- Download Binary (Recommended)

Prebuilt binaries are available in **GitHub Releases**:

👉 https://github.com/NimWithA/ton-w5-vanity/releases

Download the version for your OS and run it.

------------------------------------------------------------------------

### Option 2 --- Build From Source

``` bash
git clone https://github.com/NimWithA/ton-w5-vanity.git
cd ton-w5-vanity/ton\ vanity
go build -o ton-vanity
```

------------------------------------------------------------------------

## Usage

Run the executable:

``` bash
./ton-vanity
```

Follow the interactive prompts to configure:

-   Vanity pattern
-   Worker count
-   Network (mainnet/testnet)
-   Output file
-   Optional stats/debug

------------------------------------------------------------------------

## Output

When a match is found:

    ADDRESS: UQB...YOUR_TEXT...
    MNEMONIC: word1 word2 ... word24

Results are saved to the specified file with restricted permissions.

------------------------------------------------------------------------

## Security Notice

This tool generates **real wallet mnemonics**.

-   Anyone with access to the mnemonic controls the wallet.
-   Output is stored in plain text.
-   Do not use on untrusted systems.

------------------------------------------------------------------------

---

## Example Vanity Address

Here is an example vanity wallet address generated using this tool (contains my name **Nima** at the end):

```
UQC9ntLox55CB4k-H6iM0IFEarYA9TzvJHRKNwkQGLG_NiMA
```

If you find this project useful and would like to support its development, feel free to donate 🙏

**TON Wallet:** `UQC9ntLox55CB4k-H6iM0IFEarYA9TzvJHRKNwkQGLG_NiMA`
