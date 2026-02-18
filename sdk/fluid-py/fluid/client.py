# coding: utf-8

from __future__ import annotations

"""
Unified Fluid Client

This module provides a unified client wrapper for the Fluid SDK,
offering a cleaner interface with flattened parameters instead of request objects.

Example:
    from fluid import Fluid

    client = Fluid(host="http://localhost:8080")
    # Create a sandbox with simple parameters
    client.sandbox.create_sandbox(source_vm_name="ubuntu-base")
    # Run a command
    client.command.run_command(command="ls", args=["-la"])
"""

from typing import Dict, List, Optional, Tuple, Union

from fluid.api.access_api import AccessApi
from fluid.api.ansible_api import AnsibleApi
from fluid.api.ansible_playbooks_api import AnsiblePlaybooksApi
from fluid.api.auth_api import AuthApi
from fluid.api.billing_api import BillingApi
from fluid.api.health_api import HealthApi
from fluid.api.host_tokens_api import HostTokensApi
from fluid.api.hosts_api import HostsApi
from fluid.api.members_api import MembersApi
from fluid.api.organizations_api import OrganizationsApi
from fluid.api.sandbox_api import SandboxApi
from fluid.api.sandboxes_api import SandboxesApi
from fluid.api.source_vms_api import SourceVMsApi
from fluid.api.vms_api import VMsApi
from fluid.api_client import ApiClient
from fluid.configuration import Configuration
from fluid.models.github_com_aspectrr_fluid_sh_fluid_remote_internal_ansible_add_task_request import \
    GithubComAspectrrFluidShFluidRemoteInternalAnsibleAddTaskRequest
from fluid.models.github_com_aspectrr_fluid_sh_fluid_remote_internal_ansible_add_task_response import \
    GithubComAspectrrFluidShFluidRemoteInternalAnsibleAddTaskResponse
from fluid.models.github_com_aspectrr_fluid_sh_fluid_remote_internal_ansible_get_playbook_response import \
    GithubComAspectrrFluidShFluidRemoteInternalAnsibleGetPlaybookResponse
from fluid.models.github_com_aspectrr_fluid_sh_fluid_remote_internal_ansible_job import \
    GithubComAspectrrFluidShFluidRemoteInternalAnsibleJob
from fluid.models.github_com_aspectrr_fluid_sh_fluid_remote_internal_ansible_job_request import \
    GithubComAspectrrFluidShFluidRemoteInternalAnsibleJobRequest
from fluid.models.github_com_aspectrr_fluid_sh_fluid_remote_internal_ansible_job_response import \
    GithubComAspectrrFluidShFluidRemoteInternalAnsibleJobResponse
from fluid.models.github_com_aspectrr_fluid_sh_fluid_remote_internal_ansible_reorder_tasks_request import \
    GithubComAspectrrFluidShFluidRemoteInternalAnsibleReorderTasksRequest
from fluid.models.github_com_aspectrr_fluid_sh_fluid_remote_internal_ansible_update_task_request import \
    GithubComAspectrrFluidShFluidRemoteInternalAnsibleUpdateTaskRequest
from fluid.models.github_com_aspectrr_fluid_sh_fluid_remote_internal_ansible_update_task_response import \
    GithubComAspectrrFluidShFluidRemoteInternalAnsibleUpdateTaskResponse
from fluid.models.github_com_aspectrr_fluid_sh_fluid_remote_internal_rest_create_sandbox_request import \
    GithubComAspectrrFluidShFluidRemoteInternalRestCreateSandboxRequest
from fluid.models.github_com_aspectrr_fluid_sh_fluid_remote_internal_rest_create_sandbox_response import \
    GithubComAspectrrFluidShFluidRemoteInternalRestCreateSandboxResponse
from fluid.models.github_com_aspectrr_fluid_sh_fluid_remote_internal_rest_destroy_sandbox_response import \
    GithubComAspectrrFluidShFluidRemoteInternalRestDestroySandboxResponse
from fluid.models.github_com_aspectrr_fluid_sh_fluid_remote_internal_rest_diff_request import \
    GithubComAspectrrFluidShFluidRemoteInternalRestDiffRequest
from fluid.models.github_com_aspectrr_fluid_sh_fluid_remote_internal_rest_diff_response import \
    GithubComAspectrrFluidShFluidRemoteInternalRestDiffResponse
from fluid.models.github_com_aspectrr_fluid_sh_fluid_remote_internal_rest_discover_ip_response import \
    GithubComAspectrrFluidShFluidRemoteInternalRestDiscoverIPResponse
from fluid.models.github_com_aspectrr_fluid_sh_fluid_remote_internal_rest_get_sandbox_response import \
    GithubComAspectrrFluidShFluidRemoteInternalRestGetSandboxResponse
from fluid.models.github_com_aspectrr_fluid_sh_fluid_remote_internal_rest_inject_ssh_key_request import \
    GithubComAspectrrFluidShFluidRemoteInternalRestInjectSSHKeyRequest
from fluid.models.github_com_aspectrr_fluid_sh_fluid_remote_internal_rest_list_sandboxes_response import \
    GithubComAspectrrFluidShFluidRemoteInternalRestListSandboxesResponse
from fluid.models.github_com_aspectrr_fluid_sh_fluid_remote_internal_rest_list_vms_response import \
    GithubComAspectrrFluidShFluidRemoteInternalRestListVMsResponse
from fluid.models.github_com_aspectrr_fluid_sh_fluid_remote_internal_rest_publish_request import \
    GithubComAspectrrFluidShFluidRemoteInternalRestPublishRequest
from fluid.models.github_com_aspectrr_fluid_sh_fluid_remote_internal_rest_run_command_request import \
    GithubComAspectrrFluidShFluidRemoteInternalRestRunCommandRequest
from fluid.models.github_com_aspectrr_fluid_sh_fluid_remote_internal_rest_run_command_response import \
    GithubComAspectrrFluidShFluidRemoteInternalRestRunCommandResponse
from fluid.models.github_com_aspectrr_fluid_sh_fluid_remote_internal_rest_snapshot_request import \
    GithubComAspectrrFluidShFluidRemoteInternalRestSnapshotRequest
from fluid.models.github_com_aspectrr_fluid_sh_fluid_remote_internal_rest_snapshot_response import \
    GithubComAspectrrFluidShFluidRemoteInternalRestSnapshotResponse
from fluid.models.github_com_aspectrr_fluid_sh_fluid_remote_internal_rest_start_sandbox_response import \
    GithubComAspectrrFluidShFluidRemoteInternalRestStartSandboxResponse
from fluid.models.internal_rest_ca_public_key_response import \
    InternalRestCaPublicKeyResponse
from fluid.models.internal_rest_certificate_response import \
    InternalRestCertificateResponse
from fluid.models.internal_rest_list_certificates_response import \
    InternalRestListCertificatesResponse
from fluid.models.internal_rest_list_sessions_response import \
    InternalRestListSessionsResponse
from fluid.models.internal_rest_request_access_request import \
    InternalRestRequestAccessRequest
from fluid.models.internal_rest_request_access_response import \
    InternalRestRequestAccessResponse
from fluid.models.internal_rest_revoke_certificate_request import \
    InternalRestRevokeCertificateRequest
from fluid.models.internal_rest_revoke_certificate_response import \
    InternalRestRevokeCertificateResponse
from fluid.models.internal_rest_session_end_request import \
    InternalRestSessionEndRequest
from fluid.models.internal_rest_session_end_response import \
    InternalRestSessionEndResponse
from fluid.models.internal_rest_session_start_request import \
    InternalRestSessionStartRequest
from fluid.models.internal_rest_session_start_response import \
    InternalRestSessionStartResponse
from fluid.models.orchestrator_create_sandbox_request import \
    OrchestratorCreateSandboxRequest
from fluid.models.orchestrator_host_info import OrchestratorHostInfo
from fluid.models.orchestrator_prepare_request import \
    OrchestratorPrepareRequest
from fluid.models.orchestrator_read_source_request import \
    OrchestratorReadSourceRequest
from fluid.models.orchestrator_run_command_request import \
    OrchestratorRunCommandRequest
from fluid.models.orchestrator_run_source_request import \
    OrchestratorRunSourceRequest
from fluid.models.orchestrator_snapshot_request import \
    OrchestratorSnapshotRequest
from fluid.models.orchestrator_snapshot_response import \
    OrchestratorSnapshotResponse
from fluid.models.orchestrator_source_command_result import \
    OrchestratorSourceCommandResult
from fluid.models.orchestrator_source_file_result import \
    OrchestratorSourceFileResult
from fluid.models.rest_add_member_request import RestAddMemberRequest
from fluid.models.rest_auth_response import RestAuthResponse
from fluid.models.rest_billing_response import RestBillingResponse
from fluid.models.rest_calculator_request import RestCalculatorRequest
from fluid.models.rest_calculator_response import RestCalculatorResponse
from fluid.models.rest_create_host_token_request import \
    RestCreateHostTokenRequest
from fluid.models.rest_create_org_request import RestCreateOrgRequest
from fluid.models.rest_host_token_response import RestHostTokenResponse
from fluid.models.rest_login_request import RestLoginRequest
from fluid.models.rest_member_response import RestMemberResponse
from fluid.models.rest_org_response import RestOrgResponse
from fluid.models.rest_register_request import RestRegisterRequest
from fluid.models.rest_update_org_request import RestUpdateOrgRequest
from fluid.models.store_command import StoreCommand
from fluid.models.store_sandbox import StoreSandbox


class AccessOperations:
    """Wrapper for AccessApi with simplified method signatures."""

    def __init__(self, api: AccessApi):
        self._api = api

    def get_ca_public_key(self) -> InternalRestCaPublicKeyResponse:
        """Get the SSH CA public key

        Returns:
            InternalRestCaPublicKeyResponse: Pydantic model with full IDE autocomplete.
            Call .model_dump() to convert to dict if needed.
        """
        return self._api.get_ca_public_key()

    def get_certificate(
        self,
        cert_id: str,
    ) -> InternalRestCertificateResponse:
        """Get certificate details

        Args:
            cert_id: str

        Returns:
            InternalRestCertificateResponse: Pydantic model with full IDE autocomplete.
            Call .model_dump() to convert to dict if needed.
        """
        return self._api.get_certificate(cert_id=cert_id)

    def list_certificates(
        self,
        sandbox_id: Optional[str] = None,
        user_id: Optional[str] = None,
        status: Optional[str] = None,
        active_only: Optional[bool] = None,
        limit: Optional[int] = None,
        offset: Optional[int] = None,
    ) -> InternalRestListCertificatesResponse:
        """List certificates

        Args:
            sandbox_id: Optional[str]
            user_id: Optional[str]
            status: Optional[str]
            active_only: Optional[bool]
            limit: Optional[int]
            offset: Optional[int]

        Returns:
            InternalRestListCertificatesResponse: Pydantic model with full IDE autocomplete.
            Call .model_dump() to convert to dict if needed.
        """
        return self._api.list_certificates(
            sandbox_id=sandbox_id,
            user_id=user_id,
            status=status,
            active_only=active_only,
            limit=limit,
            offset=offset,
        )

    def list_sessions(
        self,
        sandbox_id: Optional[str] = None,
        certificate_id: Optional[str] = None,
        user_id: Optional[str] = None,
        active_only: Optional[bool] = None,
        limit: Optional[int] = None,
        offset: Optional[int] = None,
    ) -> InternalRestListSessionsResponse:
        """List sessions

        Args:
            sandbox_id: Optional[str]
            certificate_id: Optional[str]
            user_id: Optional[str]
            active_only: Optional[bool]
            limit: Optional[int]
            offset: Optional[int]

        Returns:
            InternalRestListSessionsResponse: Pydantic model with full IDE autocomplete.
            Call .model_dump() to convert to dict if needed.
        """
        return self._api.list_sessions(
            sandbox_id=sandbox_id,
            certificate_id=certificate_id,
            user_id=user_id,
            active_only=active_only,
            limit=limit,
            offset=offset,
        )

    def record_session_end(
        self,
        reason: Optional[str] = None,
        session_id: Optional[str] = None,
    ) -> InternalRestSessionEndResponse:
        """Record session end

        Args:
            reason: reason
            session_id: session_id

        Returns:
            InternalRestSessionEndResponse: Pydantic model with full IDE autocomplete.
            Call .model_dump() to convert to dict if needed.
        """
        request = InternalRestSessionEndRequest(
            reason=reason,
            session_id=session_id,
        )
        return self._api.record_session_end(request=request)

    def record_session_start(
        self,
        certificate_id: Optional[str] = None,
        source_ip: Optional[str] = None,
    ) -> InternalRestSessionStartResponse:
        """Record session start

        Args:
            certificate_id: certificate_id
            source_ip: source_ip

        Returns:
            InternalRestSessionStartResponse: Pydantic model with full IDE autocomplete.
            Call .model_dump() to convert to dict if needed.
        """
        request = InternalRestSessionStartRequest(
            certificate_id=certificate_id,
            source_ip=source_ip,
        )
        return self._api.record_session_start(request=request)

    def request_access(
        self,
        public_key: Optional[str] = None,
        sandbox_id: Optional[str] = None,
        ttl_minutes: Optional[int] = None,
        user_id: Optional[str] = None,
    ) -> InternalRestRequestAccessResponse:
        """Request SSH access to a sandbox

        Args:
            public_key: PublicKey is the user
            sandbox_id: SandboxID is the target sandbox.
            ttl_minutes: TTLMinutes is the requested access duration (1-10 minutes).
            user_id: UserID identifies the requesting user.

        Returns:
            InternalRestRequestAccessResponse: Pydantic model with full IDE autocomplete.
            Call .model_dump() to convert to dict if needed.
        """
        request = InternalRestRequestAccessRequest(
            public_key=public_key,
            sandbox_id=sandbox_id,
            ttl_minutes=ttl_minutes,
            user_id=user_id,
        )
        return self._api.request_access(request=request)

    def revoke_certificate(
        self,
        cert_id: str,
        reason: Optional[str] = None,
    ) -> InternalRestRevokeCertificateResponse:
        """Revoke a certificate

        Args:
            cert_id: str
            reason: reason

        Returns:
            InternalRestRevokeCertificateResponse: Pydantic model with full IDE autocomplete.
            Call .model_dump() to convert to dict if needed.
        """
        request = InternalRestRevokeCertificateRequest(
            reason=reason,
        )
        return self._api.revoke_certificate(cert_id=cert_id, request=request)


class AnsibleOperations:
    """Wrapper for AnsibleApi with simplified method signatures."""

    def __init__(self, api: AnsibleApi):
        self._api = api

    def create_ansible_job(
        self,
        check: Optional[bool] = None,
        playbook: Optional[str] = None,
        vm_name: Optional[str] = None,
    ) -> GithubComAspectrrFluidShFluidRemoteInternalAnsibleJobResponse:
        """Create Ansible job

        Args:
            check: check
            playbook: playbook
            vm_name: vm_name

        Returns:
            GithubComAspectrrFluidShFluidRemoteInternalAnsibleJobResponse: Pydantic model with full IDE autocomplete.
            Call .model_dump() to convert to dict if needed.
        """
        request = GithubComAspectrrFluidShFluidRemoteInternalAnsibleJobRequest(
            check=check,
            playbook=playbook,
            vm_name=vm_name,
        )
        return self._api.create_ansible_job(request=request)

    def get_ansible_job(
        self,
        job_id: str,
    ) -> GithubComAspectrrFluidShFluidRemoteInternalAnsibleJob:
        """Get Ansible job

        Args:
            job_id: str

        Returns:
            GithubComAspectrrFluidShFluidRemoteInternalAnsibleJob: Pydantic model with full IDE autocomplete.
            Call .model_dump() to convert to dict if needed.
        """
        return self._api.get_ansible_job(job_id=job_id)

    def stream_ansible_job_output(
        self,
        job_id: str,
    ) -> None:
        """Stream Ansible job output

        Args:
            job_id: str
        """
        return self._api.stream_ansible_job_output(job_id=job_id)


class AnsiblePlaybooksOperations:
    """Wrapper for AnsiblePlaybooksApi with simplified method signatures."""

    def __init__(self, api: AnsiblePlaybooksApi):
        self._api = api

    def add_playbook_task(
        self,
        playbook_name: str,
        module: Optional[str] = None,
        name: Optional[str] = None,
        params: Optional[Dict[str, Any]] = None,
    ) -> GithubComAspectrrFluidShFluidRemoteInternalAnsibleAddTaskResponse:
        """Add task to playbook

        Args:
            playbook_name: str
            module: module
            name: name
            params: params

        Returns:
            GithubComAspectrrFluidShFluidRemoteInternalAnsibleAddTaskResponse: Pydantic model with full IDE autocomplete.
            Call .model_dump() to convert to dict if needed.
        """
        request = GithubComAspectrrFluidShFluidRemoteInternalAnsibleAddTaskRequest(
            module=module,
            name=name,
            params=params,
        )
        return self._api.add_playbook_task(playbook_name=playbook_name, request=request)

    def create_playbook(
        self,
    ) -> GithubComAspectrrFluidShFluidRemoteInternalAnsibleCreatePlaybookResponse:
        """Create playbook

        Returns:
            GithubComAspectrrFluidShFluidRemoteInternalAnsibleCreatePlaybookResponse: Pydantic model with full IDE autocomplete.
            Call .model_dump() to convert to dict if needed.
        """
        request = (
            GithubComAspectrrFluidShFluidRemoteInternalAnsibleCreatePlaybookRequest()
        )
        return self._api.create_playbook(request=request)

    def delete_playbook(
        self,
        playbook_name: str,
    ) -> None:
        """Delete playbook

        Args:
            playbook_name: str
        """
        return self._api.delete_playbook(playbook_name=playbook_name)

    def delete_playbook_task(
        self,
        playbook_name: str,
        task_id: str,
    ) -> None:
        """Delete task

        Args:
            playbook_name: str
            task_id: str
        """
        return self._api.delete_playbook_task(
            playbook_name=playbook_name, task_id=task_id
        )

    def export_playbook(
        self,
        playbook_name: str,
    ) -> GithubComAspectrrFluidShFluidRemoteInternalAnsibleExportPlaybookResponse:
        """Export playbook

        Args:
            playbook_name: str

        Returns:
            GithubComAspectrrFluidShFluidRemoteInternalAnsibleExportPlaybookResponse: Pydantic model with full IDE autocomplete.
            Call .model_dump() to convert to dict if needed.
        """
        return self._api.export_playbook(playbook_name=playbook_name)

    def get_playbook(
        self,
        playbook_name: str,
    ) -> GithubComAspectrrFluidShFluidRemoteInternalAnsibleGetPlaybookResponse:
        """Get playbook

        Args:
            playbook_name: str

        Returns:
            GithubComAspectrrFluidShFluidRemoteInternalAnsibleGetPlaybookResponse: Pydantic model with full IDE autocomplete.
            Call .model_dump() to convert to dict if needed.
        """
        return self._api.get_playbook(playbook_name=playbook_name)

    def list_playbooks(
        self,
    ) -> GithubComAspectrrFluidShFluidRemoteInternalAnsibleListPlaybooksResponse:
        """List playbooks

        Returns:
            GithubComAspectrrFluidShFluidRemoteInternalAnsibleListPlaybooksResponse: Pydantic model with full IDE autocomplete.
            Call .model_dump() to convert to dict if needed.
        """
        return self._api.list_playbooks()

    def reorder_playbook_tasks(
        self,
        playbook_name: str,
        task_ids: Optional[List[str]] = None,
    ) -> None:
        """Reorder tasks

        Args:
            playbook_name: str
            task_ids: task_ids
        """
        request = GithubComAspectrrFluidShFluidRemoteInternalAnsibleReorderTasksRequest(
            task_ids=task_ids,
        )
        return self._api.reorder_playbook_tasks(
            playbook_name=playbook_name, request=request
        )

    def update_playbook_task(
        self,
        playbook_name: str,
        task_id: str,
        module: Optional[str] = None,
        name: Optional[str] = None,
        params: Optional[Dict[str, Any]] = None,
    ) -> GithubComAspectrrFluidShFluidRemoteInternalAnsibleUpdateTaskResponse:
        """Update task

        Args:
            playbook_name: str
            task_id: str
            module: module
            name: name
            params: params

        Returns:
            GithubComAspectrrFluidShFluidRemoteInternalAnsibleUpdateTaskResponse: Pydantic model with full IDE autocomplete.
            Call .model_dump() to convert to dict if needed.
        """
        request = GithubComAspectrrFluidShFluidRemoteInternalAnsibleUpdateTaskRequest(
            module=module,
            name=name,
            params=params,
        )
        return self._api.update_playbook_task(
            playbook_name=playbook_name, task_id=task_id, request=request
        )


class AuthOperations:
    """Wrapper for AuthApi with simplified method signatures."""

    def __init__(self, api: AuthApi):
        self._api = api

    def auth_github_callback_get(
        self,
        code: str,
    ) -> None:
        """GitHub OAuth callback

        Args:
            code: str
        """
        return self._api.auth_github_callback_get(code=code)

    def auth_github_get(self) -> None:
        """GitHub OAuth login"""
        return self._api.auth_github_get()

    def auth_google_callback_get(
        self,
        code: str,
    ) -> None:
        """Google OAuth callback

        Args:
            code: str
        """
        return self._api.auth_google_callback_get(code=code)

    def auth_google_get(self) -> None:
        """Google OAuth login"""
        return self._api.auth_google_get()

    def auth_login_post(
        self,
        email: Optional[str] = None,
        password: Optional[str] = None,
    ) -> RestAuthResponse:
        """Log in

        Args:
            email: email
            password: password

        Returns:
            RestAuthResponse: Pydantic model with full IDE autocomplete.
            Call .model_dump() to convert to dict if needed.
        """
        request = RestLoginRequest(
            email=email,
            password=password,
        )
        return self._api.auth_login_post(request=request)

    def auth_logout_post(self) -> Dict[str, str]:
        """Log out

        Returns:
            Dict[str, str]: Pydantic model with full IDE autocomplete.
            Call .model_dump() to convert to dict if needed.
        """
        return self._api.auth_logout_post()

    def auth_me_get(self) -> RestAuthResponse:
        """Get current user

        Returns:
            RestAuthResponse: Pydantic model with full IDE autocomplete.
            Call .model_dump() to convert to dict if needed.
        """
        return self._api.auth_me_get()

    def auth_register_post(
        self,
        display_name: Optional[str] = None,
        email: Optional[str] = None,
        password: Optional[str] = None,
    ) -> RestAuthResponse:
        """Register a new user

        Args:
            display_name: display_name
            email: email
            password: password

        Returns:
            RestAuthResponse: Pydantic model with full IDE autocomplete.
            Call .model_dump() to convert to dict if needed.
        """
        request = RestRegisterRequest(
            display_name=display_name,
            email=email,
            password=password,
        )
        return self._api.auth_register_post(request=request)


class BillingOperations:
    """Wrapper for BillingApi with simplified method signatures."""

    def __init__(self, api: BillingApi):
        self._api = api

    def billing_calculator_post(
        self,
        agent_hosts: Optional[int] = None,
        concurrent_sandboxes: Optional[int] = None,
        hours_per_month: Optional[Union[float, int]] = None,
        source_vms: Optional[int] = None,
    ) -> RestCalculatorResponse:
        """Pricing calculator

        Args:
            agent_hosts: agent_hosts
            concurrent_sandboxes: concurrent_sandboxes
            hours_per_month: hours_per_month
            source_vms: source_vms

        Returns:
            RestCalculatorResponse: Pydantic model with full IDE autocomplete.
            Call .model_dump() to convert to dict if needed.
        """
        request = RestCalculatorRequest(
            agent_hosts=agent_hosts,
            concurrent_sandboxes=concurrent_sandboxes,
            hours_per_month=hours_per_month,
            source_vms=source_vms,
        )
        return self._api.billing_calculator_post(request=request)

    def orgs_slug_billing_get(
        self,
        slug: str,
    ) -> RestBillingResponse:
        """Get billing info

        Args:
            slug: str

        Returns:
            RestBillingResponse: Pydantic model with full IDE autocomplete.
            Call .model_dump() to convert to dict if needed.
        """
        return self._api.orgs_slug_billing_get(slug=slug)

    def orgs_slug_billing_portal_post(
        self,
        slug: str,
    ) -> Dict[str, str]:
        """Billing portal

        Args:
            slug: str

        Returns:
            Dict[str, str]: Pydantic model with full IDE autocomplete.
            Call .model_dump() to convert to dict if needed.
        """
        return self._api.orgs_slug_billing_portal_post(slug=slug)

    def orgs_slug_billing_subscribe_post(
        self,
        slug: str,
    ) -> Dict[str, str]:
        """Subscribe

        Args:
            slug: str

        Returns:
            Dict[str, str]: Pydantic model with full IDE autocomplete.
            Call .model_dump() to convert to dict if needed.
        """
        return self._api.orgs_slug_billing_subscribe_post(slug=slug)

    def orgs_slug_billing_usage_get(
        self,
        slug: str,
    ) -> Dict[str, object]:
        """Get usage

        Args:
            slug: str

        Returns:
            Dict[str, object]: Pydantic model with full IDE autocomplete.
            Call .model_dump() to convert to dict if needed.
        """
        return self._api.orgs_slug_billing_usage_get(slug=slug)

    def webhooks_stripe_post(self) -> Dict[str, str]:
        """Stripe webhook

        Returns:
            Dict[str, str]: Pydantic model with full IDE autocomplete.
            Call .model_dump() to convert to dict if needed.
        """
        return self._api.webhooks_stripe_post()


class HealthOperations:
    """Wrapper for HealthApi with simplified method signatures."""

    def __init__(self, api: HealthApi):
        self._api = api

    def health_get(self) -> Dict[str, str]:
        """Health check

        Returns:
            Dict[str, str]: Pydantic model with full IDE autocomplete.
            Call .model_dump() to convert to dict if needed.
        """
        return self._api.health_get()


class HostTokensOperations:
    """Wrapper for HostTokensApi with simplified method signatures."""

    def __init__(self, api: HostTokensApi):
        self._api = api

    def orgs_slug_hosts_tokens_get(
        self,
        slug: str,
    ) -> Dict[str, object]:
        """List host tokens

        Args:
            slug: str

        Returns:
            Dict[str, object]: Pydantic model with full IDE autocomplete.
            Call .model_dump() to convert to dict if needed.
        """
        return self._api.orgs_slug_hosts_tokens_get(slug=slug)

    def orgs_slug_hosts_tokens_post(
        self,
        slug: str,
        name: Optional[str] = None,
    ) -> RestHostTokenResponse:
        """Create host token

        Args:
            slug: str
            name: name

        Returns:
            RestHostTokenResponse: Pydantic model with full IDE autocomplete.
            Call .model_dump() to convert to dict if needed.
        """
        request = RestCreateHostTokenRequest(
            name=name,
        )
        return self._api.orgs_slug_hosts_tokens_post(slug=slug, request=request)

    def orgs_slug_hosts_tokens_token_id_delete(
        self,
        slug: str,
        token_id: str,
    ) -> Dict[str, str]:
        """Delete host token

        Args:
            slug: str
            token_id: str

        Returns:
            Dict[str, str]: Pydantic model with full IDE autocomplete.
            Call .model_dump() to convert to dict if needed.
        """
        return self._api.orgs_slug_hosts_tokens_token_id_delete(
            slug=slug, token_id=token_id
        )


class HostsOperations:
    """Wrapper for HostsApi with simplified method signatures."""

    def __init__(self, api: HostsApi):
        self._api = api

    def orgs_slug_hosts_get(
        self,
        slug: str,
    ) -> Dict[str, object]:
        """List hosts

        Args:
            slug: str

        Returns:
            Dict[str, object]: Pydantic model with full IDE autocomplete.
            Call .model_dump() to convert to dict if needed.
        """
        return self._api.orgs_slug_hosts_get(slug=slug)

    def orgs_slug_hosts_host_id_get(
        self,
        slug: str,
        host_id: str,
    ) -> OrchestratorHostInfo:
        """Get host

        Args:
            slug: str
            host_id: str

        Returns:
            OrchestratorHostInfo: Pydantic model with full IDE autocomplete.
            Call .model_dump() to convert to dict if needed.
        """
        return self._api.orgs_slug_hosts_host_id_get(slug=slug, host_id=host_id)


class MembersOperations:
    """Wrapper for MembersApi with simplified method signatures."""

    def __init__(self, api: MembersApi):
        self._api = api

    def orgs_slug_members_get(
        self,
        slug: str,
    ) -> Dict[str, object]:
        """List members

        Args:
            slug: str

        Returns:
            Dict[str, object]: Pydantic model with full IDE autocomplete.
            Call .model_dump() to convert to dict if needed.
        """
        return self._api.orgs_slug_members_get(slug=slug)

    def orgs_slug_members_member_id_delete(
        self,
        slug: str,
        member_id: str,
    ) -> Dict[str, str]:
        """Remove member

        Args:
            slug: str
            member_id: str

        Returns:
            Dict[str, str]: Pydantic model with full IDE autocomplete.
            Call .model_dump() to convert to dict if needed.
        """
        return self._api.orgs_slug_members_member_id_delete(
            slug=slug, member_id=member_id
        )

    def orgs_slug_members_post(
        self,
        slug: str,
        email: Optional[str] = None,
        role: Optional[str] = None,
    ) -> RestMemberResponse:
        """Add member

        Args:
            slug: str
            email: email
            role: role

        Returns:
            RestMemberResponse: Pydantic model with full IDE autocomplete.
            Call .model_dump() to convert to dict if needed.
        """
        request = RestAddMemberRequest(
            email=email,
            role=role,
        )
        return self._api.orgs_slug_members_post(slug=slug, request=request)


class OrganizationsOperations:
    """Wrapper for OrganizationsApi with simplified method signatures."""

    def __init__(self, api: OrganizationsApi):
        self._api = api

    def orgs_get(self) -> Dict[str, object]:
        """List organizations

        Returns:
            Dict[str, object]: Pydantic model with full IDE autocomplete.
            Call .model_dump() to convert to dict if needed.
        """
        return self._api.orgs_get()

    def orgs_post(
        self,
        name: Optional[str] = None,
        slug: Optional[str] = None,
    ) -> RestOrgResponse:
        """Create organization

        Args:
            name: name
            slug: slug

        Returns:
            RestOrgResponse: Pydantic model with full IDE autocomplete.
            Call .model_dump() to convert to dict if needed.
        """
        request = RestCreateOrgRequest(
            name=name,
            slug=slug,
        )
        return self._api.orgs_post(request=request)

    def orgs_slug_delete(
        self,
        slug: str,
    ) -> Dict[str, str]:
        """Delete organization

        Args:
            slug: str

        Returns:
            Dict[str, str]: Pydantic model with full IDE autocomplete.
            Call .model_dump() to convert to dict if needed.
        """
        return self._api.orgs_slug_delete(slug=slug)

    def orgs_slug_get(
        self,
        slug: str,
    ) -> RestOrgResponse:
        """Get organization

        Args:
            slug: str

        Returns:
            RestOrgResponse: Pydantic model with full IDE autocomplete.
            Call .model_dump() to convert to dict if needed.
        """
        return self._api.orgs_slug_get(slug=slug)

    def orgs_slug_patch(
        self,
        slug: str,
        name: Optional[str] = None,
    ) -> RestOrgResponse:
        """Update organization

        Args:
            slug: str
            name: name

        Returns:
            RestOrgResponse: Pydantic model with full IDE autocomplete.
            Call .model_dump() to convert to dict if needed.
        """
        request = RestUpdateOrgRequest(
            name=name,
        )
        return self._api.orgs_slug_patch(slug=slug, request=request)


class SandboxOperations:
    """Wrapper for SandboxApi with simplified method signatures."""

    def __init__(self, api: SandboxApi):
        self._api = api

    def create_sandbox(
        self,
        agent_id: Optional[str] = None,
        auto_start: Optional[bool] = None,
        cpu: Optional[int] = None,
        memory_mb: Optional[int] = None,
        source_vm_name: Optional[str] = None,
        ttl_seconds: Optional[int] = None,
        wait_for_ip: Optional[bool] = None,
        request_timeout: Union[None, float, Tuple[float, float]] = None,
    ) -> GithubComAspectrrFluidShFluidRemoteInternalRestCreateSandboxResponse:
        """Create a new sandbox

        Args:
            agent_id: required
            auto_start: optional; if true, start the VM immediately after creation
            cpu: optional; default from service config if <=0
            memory_mb: optional; default from service config if <=0
            source_vm_name: required; name of existing VM in libvirt to clone from
            ttl_seconds: optional; TTL for auto garbage collection
            wait_for_ip: optional; if true and auto_start, wait for IP discovery. When True, consider setting request_timeout to accommodate IP discovery (server default is 120s)
            request_timeout: HTTP request timeout in seconds. Can be a single float for total timeout, or a tuple (connect_timeout, read_timeout). For operations with wait_for_ip=True, set this to at least 180 seconds.

        Returns:
            GithubComAspectrrFluidShFluidRemoteInternalRestCreateSandboxResponse: Pydantic model with full IDE autocomplete.
            Call .model_dump() to convert to dict if needed.
        """
        request = GithubComAspectrrFluidShFluidRemoteInternalRestCreateSandboxRequest(
            agent_id=agent_id,
            auto_start=auto_start,
            cpu=cpu,
            memory_mb=memory_mb,
            source_vm_name=source_vm_name,
            ttl_seconds=ttl_seconds,
            wait_for_ip=wait_for_ip,
        )
        return self._api.create_sandbox(
            request=request, _request_timeout=request_timeout
        )

    def create_snapshot(
        self,
        id: str,
        external: Optional[bool] = None,
        name: Optional[str] = None,
    ) -> GithubComAspectrrFluidShFluidRemoteInternalRestSnapshotResponse:
        """Create snapshot

        Args:
            id: str
            external: optional; default false (internal snapshot)
            name: required

        Returns:
            GithubComAspectrrFluidShFluidRemoteInternalRestSnapshotResponse: Pydantic model with full IDE autocomplete.
            Call .model_dump() to convert to dict if needed.
        """
        request = GithubComAspectrrFluidShFluidRemoteInternalRestSnapshotRequest(
            external=external,
            name=name,
        )
        return self._api.create_snapshot(id=id, request=request)

    def destroy_sandbox(
        self,
        id: str,
    ) -> GithubComAspectrrFluidShFluidRemoteInternalRestDestroySandboxResponse:
        """Destroy sandbox

        Args:
            id: str

        Returns:
            GithubComAspectrrFluidShFluidRemoteInternalRestDestroySandboxResponse: Pydantic model with full IDE autocomplete.
            Call .model_dump() to convert to dict if needed.
        """
        return self._api.destroy_sandbox(id=id)

    def diff_snapshots(
        self,
        id: str,
        from_snapshot: Optional[str] = None,
        to_snapshot: Optional[str] = None,
    ) -> GithubComAspectrrFluidShFluidRemoteInternalRestDiffResponse:
        """Diff snapshots

        Args:
            id: str
            from_snapshot: required
            to_snapshot: required

        Returns:
            GithubComAspectrrFluidShFluidRemoteInternalRestDiffResponse: Pydantic model with full IDE autocomplete.
            Call .model_dump() to convert to dict if needed.
        """
        request = GithubComAspectrrFluidShFluidRemoteInternalRestDiffRequest(
            from_snapshot=from_snapshot,
            to_snapshot=to_snapshot,
        )
        return self._api.diff_snapshots(id=id, request=request)

    def discover_sandbox_ip(
        self,
        id: str,
    ) -> GithubComAspectrrFluidShFluidRemoteInternalRestDiscoverIPResponse:
        """Discover sandbox IP

        Args:
            id: str

        Returns:
            GithubComAspectrrFluidShFluidRemoteInternalRestDiscoverIPResponse: Pydantic model with full IDE autocomplete.
            Call .model_dump() to convert to dict if needed.
        """
        return self._api.discover_sandbox_ip(id=id)

    def generate_configuration(
        self,
        id: str,
        tool: str,
    ) -> None:
        """Generate configuration

        Args:
            id: str
            tool: str
        """
        return self._api.generate_configuration(id=id, tool=tool)

    def get_sandbox(
        self,
        id: str,
        include_commands: Optional[bool] = None,
    ) -> GithubComAspectrrFluidShFluidRemoteInternalRestGetSandboxResponse:
        """Get sandbox details

        Args:
            id: str
            include_commands: Optional[bool]

        Returns:
            GithubComAspectrrFluidShFluidRemoteInternalRestGetSandboxResponse: Pydantic model with full IDE autocomplete.
            Call .model_dump() to convert to dict if needed.
        """
        return self._api.get_sandbox(id=id, include_commands=include_commands)

    def inject_ssh_key(
        self,
        id: str,
        public_key: Optional[str] = None,
        username: Optional[str] = None,
    ) -> None:
        """Inject SSH key into sandbox

        Args:
            id: str
            public_key: required
            username: required (explicit); typical:
        """
        request = GithubComAspectrrFluidShFluidRemoteInternalRestInjectSSHKeyRequest(
            public_key=public_key,
            username=username,
        )
        return self._api.inject_ssh_key(id=id, request=request)

    def list_sandbox_commands(
        self,
        id: str,
        limit: Optional[int] = None,
        offset: Optional[int] = None,
    ) -> GithubComAspectrrFluidShFluidRemoteInternalRestListSandboxCommandsResponse:
        """List sandbox commands

        Args:
            id: str
            limit: Optional[int]
            offset: Optional[int]

        Returns:
            GithubComAspectrrFluidShFluidRemoteInternalRestListSandboxCommandsResponse: Pydantic model with full IDE autocomplete.
            Call .model_dump() to convert to dict if needed.
        """
        return self._api.list_sandbox_commands(id=id, limit=limit, offset=offset)

    def list_sandboxes(
        self,
        agent_id: Optional[str] = None,
        job_id: Optional[str] = None,
        base_image: Optional[str] = None,
        state: Optional[str] = None,
        vm_name: Optional[str] = None,
        limit: Optional[int] = None,
        offset: Optional[int] = None,
    ) -> GithubComAspectrrFluidShFluidRemoteInternalRestListSandboxesResponse:
        """List sandboxes

        Args:
            agent_id: Optional[str]
            job_id: Optional[str]
            base_image: Optional[str]
            state: Optional[str]
            vm_name: Optional[str]
            limit: Optional[int]
            offset: Optional[int]

        Returns:
            GithubComAspectrrFluidShFluidRemoteInternalRestListSandboxesResponse: Pydantic model with full IDE autocomplete.
            Call .model_dump() to convert to dict if needed.
        """
        return self._api.list_sandboxes(
            agent_id=agent_id,
            job_id=job_id,
            base_image=base_image,
            state=state,
            vm_name=vm_name,
            limit=limit,
            offset=offset,
        )

    def publish_changes(
        self,
        id: str,
        job_id: Optional[str] = None,
        message: Optional[str] = None,
        reviewers: Optional[List[str]] = None,
    ) -> None:
        """Publish changes

        Args:
            id: str
            job_id: required
            message: optional commit/PR message
            reviewers: optional
        """
        request = GithubComAspectrrFluidShFluidRemoteInternalRestPublishRequest(
            job_id=job_id,
            message=message,
            reviewers=reviewers,
        )
        return self._api.publish_changes(id=id, request=request)

    def run_sandbox_command(
        self,
        id: str,
        command: Optional[str] = None,
        env: Optional[Dict[str, str]] = None,
        private_key_path: Optional[str] = None,
        timeout_sec: Optional[int] = None,
        user: Optional[str] = None,
        request_timeout: Union[None, float, Tuple[float, float]] = None,
    ) -> GithubComAspectrrFluidShFluidRemoteInternalRestRunCommandResponse:
        """Run command in sandbox

        Args:
            id: str
            command: required
            env: optional
            private_key_path: optional; if empty, uses managed credentials (requires SSH CA)
            timeout_sec: optional; default from service config
            user: optional; defaults to
            request_timeout: HTTP request timeout in seconds. Can be a single float for total timeout, or a tuple (connect_timeout, read_timeout). For operations with wait_for_ip=True, set this to at least 180 seconds.

        Returns:
            GithubComAspectrrFluidShFluidRemoteInternalRestRunCommandResponse: Pydantic model with full IDE autocomplete.
            Call .model_dump() to convert to dict if needed.
        """
        request = GithubComAspectrrFluidShFluidRemoteInternalRestRunCommandRequest(
            command=command,
            env=env,
            private_key_path=private_key_path,
            timeout_sec=timeout_sec,
            user=user,
        )
        return self._api.run_sandbox_command(
            id=id, request=request, _request_timeout=request_timeout
        )

    def start_sandbox(
        self,
        id: str,
        request_timeout: Union[None, float, Tuple[float, float]] = None,
    ) -> GithubComAspectrrFluidShFluidRemoteInternalRestStartSandboxResponse:
        """Start sandbox

        Args:
            id: str
            request_timeout: HTTP request timeout in seconds. Can be a single float for total timeout, or a tuple (connect_timeout, read_timeout). For operations with wait_for_ip=True, set this to at least 180 seconds.

        Returns:
            GithubComAspectrrFluidShFluidRemoteInternalRestStartSandboxResponse: Pydantic model with full IDE autocomplete.
            Call .model_dump() to convert to dict if needed.
        """
        return self._api.start_sandbox(id=id, _request_timeout=request_timeout)

    def stream_sandbox_activity(
        self,
        id: str,
    ) -> None:
        """Stream sandbox activity

        Args:
            id: str
        """
        return self._api.stream_sandbox_activity(id=id)


class SandboxesOperations:
    """Wrapper for SandboxesApi with simplified method signatures."""

    def __init__(self, api: SandboxesApi):
        self._api = api

    def orgs_slug_sandboxes_get(
        self,
        slug: str,
    ) -> Dict[str, object]:
        """List sandboxes

        Args:
            slug: str

        Returns:
            Dict[str, object]: Pydantic model with full IDE autocomplete.
            Call .model_dump() to convert to dict if needed.
        """
        return self._api.orgs_slug_sandboxes_get(slug=slug)

    def orgs_slug_sandboxes_post(
        self,
        slug: str,
        agent_id: Optional[str] = None,
        base_image: Optional[str] = None,
        memory_mb: Optional[int] = None,
        name: Optional[str] = None,
        network: Optional[str] = None,
        org_id: Optional[str] = None,
        source_vm: Optional[str] = None,
        ttl_seconds: Optional[int] = None,
        vcpus: Optional[int] = None,
    ) -> StoreSandbox:
        """Create sandbox

        Args:
            slug: str
            agent_id: agent_id
            base_image: base_image
            memory_mb: memory_mb
            name: name
            network: network
            org_id: org_id
            source_vm: source_vm
            ttl_seconds: ttl_seconds
            vcpus: vcpus

        Returns:
            StoreSandbox: Pydantic model with full IDE autocomplete.
            Call .model_dump() to convert to dict if needed.
        """
        request = OrchestratorCreateSandboxRequest(
            agent_id=agent_id,
            base_image=base_image,
            memory_mb=memory_mb,
            name=name,
            network=network,
            org_id=org_id,
            source_vm=source_vm,
            ttl_seconds=ttl_seconds,
            vcpus=vcpus,
        )
        return self._api.orgs_slug_sandboxes_post(slug=slug, request=request)

    def orgs_slug_sandboxes_sandbox_id_commands_get(
        self,
        slug: str,
        sandbox_id: str,
    ) -> Dict[str, object]:
        """List commands

        Args:
            slug: str
            sandbox_id: str

        Returns:
            Dict[str, object]: Pydantic model with full IDE autocomplete.
            Call .model_dump() to convert to dict if needed.
        """
        return self._api.orgs_slug_sandboxes_sandbox_id_commands_get(
            slug=slug, sandbox_id=sandbox_id
        )

    def orgs_slug_sandboxes_sandbox_id_delete(
        self,
        slug: str,
        sandbox_id: str,
    ) -> Dict[str, object]:
        """Destroy sandbox

        Args:
            slug: str
            sandbox_id: str

        Returns:
            Dict[str, object]: Pydantic model with full IDE autocomplete.
            Call .model_dump() to convert to dict if needed.
        """
        return self._api.orgs_slug_sandboxes_sandbox_id_delete(
            slug=slug, sandbox_id=sandbox_id
        )

    def orgs_slug_sandboxes_sandbox_id_get(
        self,
        slug: str,
        sandbox_id: str,
    ) -> StoreSandbox:
        """Get sandbox

        Args:
            slug: str
            sandbox_id: str

        Returns:
            StoreSandbox: Pydantic model with full IDE autocomplete.
            Call .model_dump() to convert to dict if needed.
        """
        return self._api.orgs_slug_sandboxes_sandbox_id_get(
            slug=slug, sandbox_id=sandbox_id
        )

    def orgs_slug_sandboxes_sandbox_id_run_post(
        self,
        slug: str,
        sandbox_id: str,
        command: Optional[str] = None,
        env: Optional[Dict[str, str]] = None,
        timeout_seconds: Optional[int] = None,
    ) -> StoreCommand:
        """Run command

        Args:
            slug: str
            sandbox_id: str
            command: command
            env: env
            timeout_seconds: timeout_seconds

        Returns:
            StoreCommand: Pydantic model with full IDE autocomplete.
            Call .model_dump() to convert to dict if needed.
        """
        request = OrchestratorRunCommandRequest(
            command=command,
            env=env,
            timeout_seconds=timeout_seconds,
        )
        return self._api.orgs_slug_sandboxes_sandbox_id_run_post(
            slug=slug, sandbox_id=sandbox_id, request=request
        )

    def orgs_slug_sandboxes_sandbox_id_snapshot_post(
        self,
        slug: str,
        sandbox_id: str,
        name: Optional[str] = None,
    ) -> OrchestratorSnapshotResponse:
        """Create snapshot

        Args:
            slug: str
            sandbox_id: str
            name: name

        Returns:
            OrchestratorSnapshotResponse: Pydantic model with full IDE autocomplete.
            Call .model_dump() to convert to dict if needed.
        """
        request = OrchestratorSnapshotRequest(
            name=name,
        )
        return self._api.orgs_slug_sandboxes_sandbox_id_snapshot_post(
            slug=slug, sandbox_id=sandbox_id, request=request
        )

    def orgs_slug_sandboxes_sandbox_id_start_post(
        self,
        slug: str,
        sandbox_id: str,
    ) -> Dict[str, object]:
        """Start sandbox

        Args:
            slug: str
            sandbox_id: str

        Returns:
            Dict[str, object]: Pydantic model with full IDE autocomplete.
            Call .model_dump() to convert to dict if needed.
        """
        return self._api.orgs_slug_sandboxes_sandbox_id_start_post(
            slug=slug, sandbox_id=sandbox_id
        )

    def orgs_slug_sandboxes_sandbox_id_stop_post(
        self,
        slug: str,
        sandbox_id: str,
    ) -> Dict[str, object]:
        """Stop sandbox

        Args:
            slug: str
            sandbox_id: str

        Returns:
            Dict[str, object]: Pydantic model with full IDE autocomplete.
            Call .model_dump() to convert to dict if needed.
        """
        return self._api.orgs_slug_sandboxes_sandbox_id_stop_post(
            slug=slug, sandbox_id=sandbox_id
        )

    def orgs_slug_sandboxes_sandbox_idip_get(
        self,
        slug: str,
        sandbox_id: str,
    ) -> Dict[str, object]:
        """Get sandbox IP

        Args:
            slug: str
            sandbox_id: str

        Returns:
            Dict[str, object]: Pydantic model with full IDE autocomplete.
            Call .model_dump() to convert to dict if needed.
        """
        return self._api.orgs_slug_sandboxes_sandbox_idip_get(
            slug=slug, sandbox_id=sandbox_id
        )


class SourceVMsOperations:
    """Wrapper for SourceVMsApi with simplified method signatures."""

    def __init__(self, api: SourceVMsApi):
        self._api = api

    def orgs_slug_sources_vm_prepare_post(
        self,
        slug: str,
        vm: str,
        ssh_key_path: Optional[str] = None,
        ssh_user: Optional[str] = None,
    ) -> Dict[str, object]:
        """Prepare source VM

        Args:
            slug: str
            vm: str
            ssh_key_path: ssh_key_path
            ssh_user: ssh_user

        Returns:
            Dict[str, object]: Pydantic model with full IDE autocomplete.
            Call .model_dump() to convert to dict if needed.
        """
        request = OrchestratorPrepareRequest(
            ssh_key_path=ssh_key_path,
            ssh_user=ssh_user,
        )
        return self._api.orgs_slug_sources_vm_prepare_post(
            slug=slug, vm=vm, request=request
        )

    def orgs_slug_sources_vm_read_post(
        self,
        slug: str,
        vm: str,
        path: Optional[str] = None,
    ) -> OrchestratorSourceFileResult:
        """Read source file

        Args:
            slug: str
            vm: str
            path: path

        Returns:
            OrchestratorSourceFileResult: Pydantic model with full IDE autocomplete.
            Call .model_dump() to convert to dict if needed.
        """
        request = OrchestratorReadSourceRequest(
            path=path,
        )
        return self._api.orgs_slug_sources_vm_read_post(
            slug=slug, vm=vm, request=request
        )

    def orgs_slug_sources_vm_run_post(
        self,
        slug: str,
        vm: str,
        command: Optional[str] = None,
        timeout_seconds: Optional[int] = None,
    ) -> OrchestratorSourceCommandResult:
        """Run source command

        Args:
            slug: str
            vm: str
            command: command
            timeout_seconds: timeout_seconds

        Returns:
            OrchestratorSourceCommandResult: Pydantic model with full IDE autocomplete.
            Call .model_dump() to convert to dict if needed.
        """
        request = OrchestratorRunSourceRequest(
            command=command,
            timeout_seconds=timeout_seconds,
        )
        return self._api.orgs_slug_sources_vm_run_post(
            slug=slug, vm=vm, request=request
        )

    def orgs_slug_vms_get(
        self,
        slug: str,
    ) -> Dict[str, object]:
        """List source VMs

        Args:
            slug: str

        Returns:
            Dict[str, object]: Pydantic model with full IDE autocomplete.
            Call .model_dump() to convert to dict if needed.
        """
        return self._api.orgs_slug_vms_get(slug=slug)


class VMsOperations:
    """Wrapper for VMsApi with simplified method signatures."""

    def __init__(self, api: VMsApi):
        self._api = api

    def list_virtual_machines(
        self,
    ) -> GithubComAspectrrFluidShFluidRemoteInternalRestListVMsResponse:
        """List all host VMs

        Returns:
            GithubComAspectrrFluidShFluidRemoteInternalRestListVMsResponse: Pydantic model with full IDE autocomplete.
            Call .model_dump() to convert to dict if needed.
        """
        return self._api.list_virtual_machines()


class Fluid:
    """Unified client for the Fluid API.

    This class provides a single entry point for all Fluid API operations.
    All methods use flattened parameters instead of request objects.

    Args:
        host: Base URL for the main Fluid API
        api_key: Optional API key for authentication
        verify_ssl: Whether to verify SSL certificates

    Example:
        >>> from fluid import Fluid
        >>> client = Fluid()
        >>> client.sandbox.create_sandbox(source_vm_name="base-vm")
    """

    def __init__(
        self,
        host: str = "http://localhost:8080",
        api_key: Optional[str] = None,
        access_token: Optional[str] = None,
        username: Optional[str] = None,
        password: Optional[str] = None,
        verify_ssl: bool = True,
        ssl_ca_cert: Optional[str] = None,
        retries: Optional[int] = None,
    ) -> None:
        """Initialize the Fluid client."""
        self._main_config = Configuration(
            host=host,
            api_key={"Authorization": api_key} if api_key else None,
            access_token=access_token,
            username=username,
            password=password,
            ssl_ca_cert=ssl_ca_cert,
            retries=retries,
        )
        self._main_config.verify_ssl = verify_ssl
        self._main_api_client = ApiClient(configuration=self._main_config)

        self._access: Optional[AccessOperations] = None
        self._ansible: Optional[AnsibleOperations] = None
        self._ansible_playbooks: Optional[AnsiblePlaybooksOperations] = None
        self._auth: Optional[AuthOperations] = None
        self._billing: Optional[BillingOperations] = None
        self._health: Optional[HealthOperations] = None
        self._host_tokens: Optional[HostTokensOperations] = None
        self._hosts: Optional[HostsOperations] = None
        self._members: Optional[MembersOperations] = None
        self._organizations: Optional[OrganizationsOperations] = None
        self._sandbox: Optional[SandboxOperations] = None
        self._sandboxes: Optional[SandboxesOperations] = None
        self._source_vms: Optional[SourceVMsOperations] = None
        self._vms: Optional[VMsOperations] = None

    @property
    def access(self) -> AccessOperations:
        """Access AccessApi operations."""
        if self._access is None:
            api = AccessApi(api_client=self._main_api_client)
            self._access = AccessOperations(api)
        return self._access

    @property
    def ansible(self) -> AnsibleOperations:
        """Access AnsibleApi operations."""
        if self._ansible is None:
            api = AnsibleApi(api_client=self._main_api_client)
            self._ansible = AnsibleOperations(api)
        return self._ansible

    @property
    def ansible_playbooks(self) -> AnsiblePlaybooksOperations:
        """Access AnsiblePlaybooksApi operations."""
        if self._ansible_playbooks is None:
            api = AnsiblePlaybooksApi(api_client=self._main_api_client)
            self._ansible_playbooks = AnsiblePlaybooksOperations(api)
        return self._ansible_playbooks

    @property
    def auth(self) -> AuthOperations:
        """Access AuthApi operations."""
        if self._auth is None:
            api = AuthApi(api_client=self._main_api_client)
            self._auth = AuthOperations(api)
        return self._auth

    @property
    def billing(self) -> BillingOperations:
        """Access BillingApi operations."""
        if self._billing is None:
            api = BillingApi(api_client=self._main_api_client)
            self._billing = BillingOperations(api)
        return self._billing

    @property
    def health(self) -> HealthOperations:
        """Access HealthApi operations."""
        if self._health is None:
            api = HealthApi(api_client=self._main_api_client)
            self._health = HealthOperations(api)
        return self._health

    @property
    def host_tokens(self) -> HostTokensOperations:
        """Access HostTokensApi operations."""
        if self._host_tokens is None:
            api = HostTokensApi(api_client=self._main_api_client)
            self._host_tokens = HostTokensOperations(api)
        return self._host_tokens

    @property
    def hosts(self) -> HostsOperations:
        """Access HostsApi operations."""
        if self._hosts is None:
            api = HostsApi(api_client=self._main_api_client)
            self._hosts = HostsOperations(api)
        return self._hosts

    @property
    def members(self) -> MembersOperations:
        """Access MembersApi operations."""
        if self._members is None:
            api = MembersApi(api_client=self._main_api_client)
            self._members = MembersOperations(api)
        return self._members

    @property
    def organizations(self) -> OrganizationsOperations:
        """Access OrganizationsApi operations."""
        if self._organizations is None:
            api = OrganizationsApi(api_client=self._main_api_client)
            self._organizations = OrganizationsOperations(api)
        return self._organizations

    @property
    def sandbox(self) -> SandboxOperations:
        """Access SandboxApi operations."""
        if self._sandbox is None:
            api = SandboxApi(api_client=self._main_api_client)
            self._sandbox = SandboxOperations(api)
        return self._sandbox

    @property
    def sandboxes(self) -> SandboxesOperations:
        """Access SandboxesApi operations."""
        if self._sandboxes is None:
            api = SandboxesApi(api_client=self._main_api_client)
            self._sandboxes = SandboxesOperations(api)
        return self._sandboxes

    @property
    def source_vms(self) -> SourceVMsOperations:
        """Access SourceVMsApi operations."""
        if self._source_vms is None:
            api = SourceVMsApi(api_client=self._main_api_client)
            self._source_vms = SourceVMsOperations(api)
        return self._source_vms

    @property
    def vms(self) -> VMsOperations:
        """Access VMsApi operations."""
        if self._vms is None:
            api = VMsApi(api_client=self._main_api_client)
            self._vms = VMsOperations(api)
        return self._vms

    @property
    def configuration(self) -> Configuration:
        """Get the main API configuration."""
        return self._main_config

    def set_debug(self, debug: bool) -> None:
        """Enable or disable debug mode."""
        self._main_config.debug = debug

    def close(self) -> None:
        """Close the API client connections."""
        if hasattr(self._main_api_client.rest_client, "close"):
            self._main_api_client.rest_client.close()

    def __enter__(self) -> "Fluid":
        """Context manager entry."""
        return self

    def __exit__(self, exc_type, exc_val, exc_tb) -> None:
        """Context manager exit."""
        self.close()
