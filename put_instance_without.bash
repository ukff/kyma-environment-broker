# ./internal/broker/broker.go:13: KymaServiceID   = "47c9dcbf-ff30-448e-ab36-d3bad66ba281"
# ./internal/broker/plans.go:19:  AWSPlanID         = "361c511f-f939-4621-b228-d0fb79a1fe15"
# trial 7d55d31d-35ae-4438-bf13-6ffdfa107d9f

instance=lj-t03
curl -XPUT "localhost:8080/oauth/v2/service_instances/$instance?accepts_incomplete=true" -i \
    -H "X-Broker-API-Version: 2.14" \
    -H "Content-Type: application/json" \
    --data-binary @- << EOF
{
   "service_id":"47c9dcbf-ff30-448e-ab36-d3bad66ba281",
   "plan_id":"7d55d31d-35ae-4438-bf13-6ffdfa107d9f",
   "context":{
       "globalaccount_id":"e449f875-b5b2-4485-b7c0-98725c0571bf",
       "subaccount_id":"19ea2000-865d-42fd-8d6f-3e3e2dcce3c8",
       "user_id":"marek.michali@sap.com"
   },
   "parameters":{
       "name":"$instance",
			  "modules": {
				  "list": []
				}
   }
}
EOF

#while ! wget https://kyma-env-broker.cp.dev.kyma.cloud.sap/kubeconfig/$instance > /dev/null 2>&1 ; do
#  sleep 30
#done
#echo "export KUBECONFIG=`pwd`/$instance"
