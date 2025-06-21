#!/bin/bash

# Sourced from scenario.sh and uses functions defined there.

base_name="rhel96-bootc-source"

scenario_create_vms() {
    prepare_kickstart host1 kickstart-bootc.ks.template "${base_name}-base"
    launch_vm --boot_blueprint rhel96-bootc
}

scenario_remove_vms() {
    remove_vm host1
}

scenario_run_tests() {
    prepare_static_delta "${base_name}-base" "${base_name}"
    consume_static_delta "${base_name}-base" "${base_name}"

    prepare_static_delta "${base_name}" "${base_name}-optionals"
    consume_static_delta "${base_name}" "${base_name}-optionals"
    
    for ref in "${base_name}-patched" "${base_name}-optionals-patched" ; do
        run_tests host1 \
            --variable "TARGET_REF:${ref}" \
            --variable "BOOTC_REGISTRY:${MIRROR_REGISTRY_URL}" \
            suites/upgrade/upgrade-successful.robot
    done
}
