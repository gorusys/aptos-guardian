package macros

const (
	TopicGas       = "gas"
	TopicStaking   = "staking"
	TopicSwitchRPC = "switch_rpc"
	TopicScam      = "scam"
)

var fixMacros = map[string]string{
	TopicGas:       "**Gas fees:** Ensure you have enough APT for gas. Retry during low network congestion. If the transaction fails, wait a few minutes and try again.",
	TopicStaking:   "**Staking unlock:** Use the same wallet you staked with. Unlock period must complete before you can withdraw. Check the staking dashboard for the exact unlock time.",
	TopicSwitchRPC: "**Switch RPC:** Use the recommended RPC from `/status` or `/rpc`. In your wallet or dApp settings, replace the current RPC URL with a healthy provider (e.g. Aptos Labs fullnode: `https://fullnode.mainnet.aptoslabs.com/v1`).",
	TopicScam:      "**Scam safety:** Mods and admins never DM you first. Official support is only in this server's channels. Never share your seed phrase or private keys. If someone DMs you claiming to be support, it's a scam.",
}

func FixContent(topic string) string {
	if topic == "" {
		return "Usage: `/fix <topic>`. Topics: `gas`, `staking`, `switch_rpc`, `scam`."
	}
	if content, ok := fixMacros[topic]; ok {
		return content
	}
	return "Unknown topic. Use one of: `gas`, `staking`, `switch_rpc`, `scam`."
}

func AllFixTopics() []string {
	return []string{TopicGas, TopicStaking, TopicSwitchRPC, TopicScam}
}
