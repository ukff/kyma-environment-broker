package globalaccounts

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/kyma-project/kyma-environment-broker/internal/events"
	"github.com/kyma-project/kyma-environment-broker/internal/k8s"
	"github.com/kyma-project/kyma-environment-broker/internal/storage"
	"github.com/kyma-project/kyma-environment-broker/internal/storage/dbmodel"
	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2/clientcredentials"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	k8scfg "sigs.k8s.io/controller-runtime/pkg/client/config"
)

const (
	dbPort  = 5432
	kebPort = 8080
)

type pair struct {
	SubaccountID    string
	GlobalAccountID string
}

type result struct {
	GlobalAccountGUID string `json:"globalAccountGUID"`
}

func (p *pair) print() {
	fmt.Printf("Global account id: %s, Subaccount id: %s\n", p.GlobalAccountID, p.SubaccountID)
}

type svcConfig struct {
	ClientID       string
	ClientSecret   string
	AuthURL        string
	SubaccountsURL string
}

func Run(c Config) {
	ctx := context.Background()
	logs := logrus.New()

	svcConfig := svcConfig{
		ClientID:     c.AccountServiceID,     // cis-creds-accounts id -> secret
		ClientSecret: c.AccountServiceSecret, // cis-creds-accounts key -> secret
		AuthURL:      c.AccountServiceURL,
	}

	oauthConfig := clientcredentials.Config{
		ClientID:     svcConfig.ClientID,
		ClientSecret: svcConfig.ClientSecret,
		TokenURL:     svcConfig.AuthURL,
	}

	db, connection, err := storage.NewFromConfig(
		c.Database,
		events.Config{},
		storage.NewEncrypter(c.Database.SecretKey),
		logs.WithField("service", "storage"))

	//time.Sleep(time.Second * 10)
	defer func() {
		err = connection.Close()
		if err != nil {
			fmt.Println(err)
		}
	}()

	kcpK8sClient, err := getKcpClient()
	fatalOnError(err, logs)

	gvk, err := k8s.GvkByName(k8s.KymaCr)
	fatalOnError(err, logs)

	svc := oauthConfig.Client(ctx)

	kymas := unstructured.UnstructuredList{}
	kymas.SetGroupVersionKind(gvk)
	err = kcpK8sClient.List(ctx, &kymas)
	fatalOnError(err, logs)

	var data strings.Builder
	for _, kyma := range kymas.Items {
		runtimeId := kyma.GetName() // name of kyma is runtime id
		runtimeIDFilter := dbmodel.InstanceFilter{RuntimeIDs: []string{runtimeId}}
		instances, _, _, err := db.Instances().List(runtimeIDFilter)
		if err != nil {
			logs.Println(err)
			continue
		}
		if len(instances) > 1 {
			logs.Errorf("more than one instance for runtime id %s", runtimeId)
			continue
		}
		if len(instances) == 0 {
			logs.Errorf("no instance for runtime id %s", runtimeId)
			continue
		}

		instance := instances[0]
		if instance.SubAccountID == "" {
			logs.Errorf("instance have empty SA %s", instance.SubAccountID)
			continue
		}
		if instance.GlobalAccountID == "" {
			logs.Errorf("instance have empty GA %s", instance.GlobalAccountID)
			continue
		}
		request, err := http.NewRequest(http.MethodGet, fmt.Sprintf(c.AccountServiceURL, instance.SubAccountID), nil)
		if err != nil {
			logs.Errorf("error creating request %s", err)
			continue
		}
		query := request.URL.Query()
		request.URL.RawQuery = query.Encode()
		response, err := svc.Do(request)
		defer func() {
			err = response.Body.Close()
			if err != nil {
				logs.Error(err)
			}
		}()

		var svcResponse result
		err = json.NewDecoder(response.Body).Decode(&svcResponse)
		if err != nil {
			logs.Error(err.Error())
			continue
		}
		log := ""
		switch {
		case svcResponse.GlobalAccountGUID == "":
			log = fmt.Sprintf(" [EMPTY] for SubAccount %s -> GA ID in KEB %s GA ID in SVC %s \n", instance.SubAccountID, instance.GlobalAccountID, svcResponse.GlobalAccountGUID)
		case svcResponse.GlobalAccountGUID != instance.GlobalAccountID:
			log = fmt.Sprintf(" [WRONG] for SubAccount %s -> GA ID in KEB %s GA ID in SVC %s \n", instance.SubAccountID, instance.GlobalAccountID, svcResponse.GlobalAccountGUID)
		default:
			log = fmt.Sprintf(" [OK] for SubAccount %s -> GA ID in KEB %s GA ID in SVC %s \n", instance.SubAccountID, instance.GlobalAccountID, svcResponse.GlobalAccountGUID)
		}
		data.WriteString(log)
	}

	logs.Info(data.String())
}

func getKcpClient() (client.Client, error) {
	kcpK8sConfig, err := k8scfg.GetConfig()
	mapper, err := apiutil.NewDiscoveryRESTMapper(kcpK8sConfig)
	if err != nil {
		err = wait.Poll(time.Second, time.Minute, func() (bool, error) {
			mapper, err = apiutil.NewDiscoveryRESTMapper(kcpK8sConfig)
			if err != nil {
				return false, nil
			}
			return true, nil
		})
		if err != nil {
			return nil, fmt.Errorf("while waiting for client mapper: %w", err)
		}
	}
	cli, err := client.New(kcpK8sConfig, client.Options{Mapper: mapper})
	if err != nil {
		return nil, fmt.Errorf("while creating a client: %w", err)
	}
	return cli, nil
}

func fatalOnError(err error, log logrus.FieldLogger) {
	if err != nil {
		log.Fatal(err)
	}
}
