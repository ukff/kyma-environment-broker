
metric name| type | labels | description
------------------------------------------------|------|--------|------------
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