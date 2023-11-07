# ./internal/broker/broker.go:13: KymaServiceID   = "47c9dcbf-ff30-448e-ab36-d3bad66ba281"
# ./internal/broker/plans.go:19:  AWSPlanID         = "361c511f-f939-4621-b228-d0fb79a1fe15"
# trial 7d55d31d-35ae-4438-bf13-6ffdfa107d9f

curl -XPUT "localhost:8080/oauth/v2/service_instances/lj-filled-list?accepts_incomplete=true" -i \
    -H "X-Broker-API-Version: 2.14" \
    -H "Content-Type: application/json" \
    --data-binary @- << EOF
{
   "service_id":"47c9dcbf-ff30-448e-ab36-d3bad66ba281",
   "plan_id":"7d55d31d-35ae-4438-bf13-6ffdfa107d9f",
   "context":{
       "globalaccount_id":"e449f875-b5b2-4485-b7c0-98725c0571bf",
       "subaccount_id":"4c0c95c3-a810-448f-818d-68ca12f62140",
       "user_id":"lukasz.jezak@sap.com"
   },
   "parameters":{
       "name":"lj-filled-list",
       "modules": {
            "list" : [
                {
                    "name" : "btp-operator"
                },
                {
                    "name" : "keda",
                    "channel" : "regular"
                }
            ]
       }
   }
}
EOF