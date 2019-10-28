// Copyright © 2019 Vulcanize, Inc
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

package cmd

import (
	"fmt"

	"github.com/ethereum/go-ethereum/rpc"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"github.com/vulcanize/vulcanizedb/pkg/geth/client"

	"github.com/spf13/cobra"
)

// testClientCmd represents the testClient command
var testClientCmd = &cobra.Command{
	Use:   "testClient",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		subCommand = cmd.CalledAs()
		logWithCommand = *log.WithField("SubCommand", subCommand)
		testClient()
	},
}

func init() {
	rootCmd.AddCommand(testClientCmd)
}

func testClient() {
	rawRPCClient, err := rpc.Dial(ipc)

	if err != nil {
		logWithCommand.Fatal(err)
	}
	rpcClient := client.NewRpcClient(rawRPCClient, ipc)
	var archivalRpcClient client.RpcClient
	if viper.GetBool("superNodeBackFill.on") && viper.GetString("superNodeBackFill.rpcPath") != "" {
		rawRPCClient, dialErr := rpc.Dial(viper.GetString("superNodeBackFill.rpcPath"))
		if dialErr != nil {
			logWithCommand.Fatal(dialErr)
		}
		archivalRpcClient = client.NewRpcClient(rawRPCClient, ipc)
	}
	mods, err := rpcClient.SupportedModules()
	if err != nil {
		logWithCommand.Fatal(err)
	}
	fmt.Printf("full node modules: %v\r\n", mods)
	archivalMods, err := archivalRpcClient.SupportedModules()
	if err != nil {
		logWithCommand.Fatal(err)
	}
	fmt.Printf("archival node modules: %v\r\n", archivalMods)
}
