# dea CLI

Command-line interface for the dea-exmachina agent system.

Communicates with Edge Function endpoints using scoped workspace JWTs.

## Install

```bash
curl -fsSL https://raw.githubusercontent.com/dea-exmachina/dea-cli/main/install.sh | sh
```

Or via Homebrew:

```bash
brew install dea-exmachina/tap/dea
```

## Usage

```bash
# Authenticate
dea auth login

# Pull assigned cards
dea pull

# Claim a card
dea claim <card-id>

# Transition a card
dea transition <card-id> --lane in_progress

# Mark done
dea done <card-id>

# Submit artifact
dea artifact <card-id> --file output.md

# Emit learning signal
dea signal <card-id> --domain engineering --pattern "..."

# Auto mode (pull → claim → work loop)
dea auto
```

## License

MIT
