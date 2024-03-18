KEB metrics are exposed via the /metrics endpoint, which is called by prometheus in job manner every given duration of time and gather the metrics which are later pushed to Victoria metrics and Plutno dashboard and alerts.

Metrics are counted in KEB memory, by two main ways, one of it is to pull data from database and update inmemory metric, or by events send across systems from business processes to pubsub, which handle event.

![KEB metrics](../assets/metrics.svg)

metric name| type | labels | 
------------------------------------------------|------|--------
kcp_keb_v2_ers_context_license_type_total | gauge | license_type | 
kcp_keb_v2_global_account_id_instances_total | gauge | global_account_id |
kcp_keb_v2_instances_total | gauge | - |
kcp_keb_v2_deprovisioning_duration_minutes | histogram | plan_id |
kcp_keb_v2_provisioning_duration_minutes | histogram | plan_id |
kcp_keb_v2_operation_result | gauge | operation_id, instance_id, global_account_id, plan_id, type, state, error_category, error_reason, error
kcp_keb_v2_operations_provisioning_failed_total | counter | plan_id
kcp_keb_v2_operations_provisioning_in_progress_total | gauge | plan_id
kcp_keb_v2_operations_provisioning_succeeded_total | counter | plan_id
kcp_keb_v2_operations_deprovisioning_failed_total | counter | plan_id
kcp_keb_v2_operations_deprovisioning_in_progress_total | gauge | plan_id
kcp_keb_v2_operations_deprovisioning_succeeded_total | counter | plan_id
kcp_keb_v2_operations_update_failed_total | counter | plan_id
kcp_keb_v2_operations_update_in_progress_total | gauge | plan_id
kcp_keb_v2_operations_update_succeeded_total | counter | plan_id

