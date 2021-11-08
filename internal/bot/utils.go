package bot

import (
	"strings"
)

// Truncate very big message
func truncateMessage(str string) string {
	truncateMsg := str
	if len(str) > 4095 { // telegram API can only support 4096 bytes per message
		// log.Warn("msg", "Message is bigger than 4095, truncate...")

		// find the end of last alert, we do not want break the html tags
		i := strings.LastIndex(str[0:4080], "\n\n") // 4080 + "\n<b>[SNIP]</b>" == 4095
		if i > 1 {
			truncateMsg = str[0:i] + "\n<b>[SNIP]</b>"
		} else {
			truncateMsg = "Message is too long... can't send.."

			// log.Warn("msg", "truncateMessage: Unable to find the end of last alert.")
		}

		return truncateMsg
	}

	return truncateMsg
}
