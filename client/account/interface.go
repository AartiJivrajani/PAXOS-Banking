package account

import (
	"PAXOS-Banking/common"
	"os"
	"strconv"

	log "github.com/Sirupsen/logrus"
	"github.com/manifoldco/promptui"
)

func (client *Client) StartTransactions() {
	var (
		err                       error
		receiverClient, amountStr string
		amount                    int
		transactionType           string
		receiverClientId          int
	)
	for {
		prompt := promptui.Select{
			Label: "Select Transaction",
			Items: []string{"Transaction", "Show Log", "Show Blockchain", "Show Balance", "Exit"},
		}

		_, transactionType, err = prompt.Run()
		if err != nil {
			log.WithFields(log.Fields{
				"error": err.Error(),
			}).Error("error fetching transaction type from the command line")
			continue
		}
		log.WithFields(log.Fields{
			"choice": transactionType,
		}).Debug("You choose...")
		switch transactionType {
		case "Exit":
			log.Debug("Fun doing business with you, see you soon!")
			os.Exit(0)

		case "Show Balance":
			message := &common.Message{
				Type: transactionType,
			}
			SendRequestToServer(message)
		case "Transfer":
			prompt := promptui.Prompt{
				Label: "Receiver Client",
			}
			receiverClient, err = prompt.Run()
			if err != nil {
				log.WithFields(log.Fields{
					"error": err.Error(),
				}).Error("error fetching the client number from the command line")
				continue
			}
			receiverClientId, _ = strconv.Atoi(receiverClient)
			if receiverClientId == client.Id {
				log.Error("you cant send money to yourself!")
				continue
			}
			prompt = promptui.Prompt{
				Label:   "Amount to be transacted",
				Default: "",
			}
			amountStr, err = prompt.Run()
			if err != nil {
				log.WithFields(log.Fields{
					"error": err.Error(),
				}).Error("error fetching the transaction amount from the command line")
				continue
			}
			amount, _ = strconv.Atoi(amountStr)
			SendRequestToServer(&common.Message{
				Type: common.TRANSACTION_MESSAGE,
				TxnMessage: &common.TransferTxn{
					Sender: client.Id,
					Recvr:  receiverClientId,
					Amount: amount,
				},
			})
		}
		//<-showNextPrompt
	}
}
