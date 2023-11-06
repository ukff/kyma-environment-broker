# ./internal/broker/broker.go:13: KymaServiceID   = "47c9dcbf-ff30-448e-ab36-d3bad66ba281"
# ./internal/broker/plans.go:19:  AWSPlanID         = "361c511f-f939-4621-b228-d0fb79a1fe15"
# ./internal/broker/plans.go:19:  AzurePlanID         = "4deee563-e5ec-4731-b9b1-53b42d855f0c"

instance=lj-1
curl -XDELETE "localhost:8080/oauth/v2/service_instances/${instance}?service_id=47c9dcbf-ff30-448e-ab36-d3bad66ba281&plan_id=4deee563-e5ec-4731-b9b1-53b42d855f0c" -i \
    -H "X-Broker-API-Version: 2.14" -H "User-Agent: accountcleanup-job"
