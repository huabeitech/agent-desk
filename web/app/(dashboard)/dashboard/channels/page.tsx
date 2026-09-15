"use client"

import {
  createDashboardStatusColumn,
  DashboardCrudPage,
} from "@/components/dashboard/crud"
import { Badge } from "@/components/ui/badge"
import {
  createChannel,
  deleteChannel,
  fetchChannels,
  updateChannel,
  updateChannelStatus,
  type AdminChannel,
  type CreateAdminChannelPayload,
} from "@/lib/api/admin"
import { ChannelTypeIcon, getChannelTypeLabel } from "@/lib/channel-i18n"
import { getEnumOptions } from "@/lib/enums"
import { Status, StatusLabels } from "@/lib/generated/enums"
import { useI18n } from "@/i18n/provider"
import { EditDialog } from "./_components/edit"

function getStatusLabel(status: Status, t: (key: string) => string) {
  if (status === Status.Disabled) {
    return t("status.disabled")
  }
  if (status === Status.Deleted) {
    return t("status.deleted")
  }
  return t("status.ok")
}

export default function DashboardChannelsPage() {
  const t = useI18n()
  const statusOptions = [
    { value: "all", label: t("status.all") },
    ...getEnumOptions(StatusLabels).map((option) => ({
      value: String(option.value),
      label: getStatusLabel(option.value as Status, t),
    })),
  ]
  const channelTypeOptions = [
    { value: "all", label: t("channel.allTypes") },
    { value: "web", label: t("channel.typeWeb") },
    { value: "telegram", label: t("channel.typeTelegram") },
    { value: "zalo_oa", label: t("channel.typeZaloOa") },
    { value: "wechat_mp", label: t("channel.typeWechatMp") },
    { value: "wxwork_kf", label: t("channel.typeWxworkKf") },
  ]

  return (
    <DashboardCrudPage<AdminChannel, CreateAdminChannelPayload>
      filters={[
        {
          name: "name",
          label: t("channel.filterName"),
          placeholder: t("channel.filterName"),
          defaultValue: "",
          trim: true,
          className: "w-full sm:w-56",
        },
        {
          name: "channelId",
          label: t("channel.filterChannelId"),
          placeholder: t("channel.filterChannelId"),
          defaultValue: "",
          trim: true,
          className: "w-full sm:w-72",
        },
        {
          name: "channelType",
          label: t("channel.allTypes"),
          type: "select",
          defaultValue: "all",
          allValue: "all",
          options: channelTypeOptions,
          className: "w-full sm:w-40",
        },
        {
          name: "status",
          label: t("status.all"),
          type: "select",
          defaultValue: "all",
          allValue: "all",
          options: statusOptions,
          className: "w-full sm:w-36",
        },
      ]}
      columns={[
        {
          key: "channel",
          label: t("channel.columnChannel"),
          render: (item) => (
            <div className="flex items-center gap-3">
              <div className="flex size-10 items-center justify-center rounded-md bg-muted">
                <ChannelTypeIcon channelType={item.channelType} className="size-4" />
              </div>
              <div>
                <div className="font-medium">{item.name}</div>
                <div className="text-xs text-muted-foreground">
                  {getChannelTypeLabel(item.channelType, t)}
                </div>
              </div>
            </div>
          ),
        },
        {
          key: "type",
          label: t("channel.columnType"),
          render: (item) => (
            <Badge variant="outline">
              {getChannelTypeLabel(item.channelType, t)}
            </Badge>
          ),
        },
        {
          key: "channelId",
          label: "ChannelID",
          render: (item) => (
            <span className="font-mono text-xs">{item.channelId || "-"}</span>
          ),
        },
        {
          key: "agent",
          label: t("channel.columnAgent"),
          render: (item) => item.aiAgentName || "-",
        },
        createDashboardStatusColumn<AdminChannel, Status>({
          label: t("channel.columnStatus"),
          getStatus: (item) => item.status as Status,
          getLabel: (status) => getStatusLabel(status, t),
          getBadgeVariant: (status) => (status === Status.Ok ? "default" : "outline"),
          isEnabled: (status) => status === Status.Ok,
          toggle: {
            getNextStatus: (item) =>
              item.status === Status.Ok ? Status.Disabled : Status.Ok,
            updateStatus: (item, nextStatus) =>
              updateChannelStatus(item.id, nextStatus),
            successMessage: (item, nextStatus) =>
              t(
                nextStatus === Status.Ok
                  ? "channel.statusEnabled"
                  : "channel.statusDisabled",
                { name: item.name }
              ),
            errorMessage: t("channel.statusUpdateFailed"),
            ariaLabel: (item) => t("channel.toggleStatus", { name: item.name }),
          },
        }),
      ]}
      fetchList={fetchChannels}
      getItemId={(item) => item.id}
      createItem={createChannel}
      updateItem={(item, payload) => updateChannel({ id: item.id, ...payload })}
      deleteItem={(item) => deleteChannel(item.id)}
      renderEditDialog={({ open, saving, itemId, onOpenChange, onSubmit }) => (
        <EditDialog
          open={open}
          saving={saving}
          itemId={itemId}
          onOpenChange={onOpenChange}
          onSubmit={onSubmit}
        />
      )}
      labels={{
        refresh: t("channel.refresh"),
        create: t("channel.new"),
        query: t("channel.query"),
        loading: t("channel.loading"),
        empty: t("channel.empty"),
        actions: t("channel.columnActions"),
        edit: t("channel.edit"),
        delete: t("channel.delete"),
        processing: t("channel.processing"),
        moreActions: (item) => t("channel.moreActions", { name: item.name }),
        loadFailed: t("channel.loadFailed"),
        saveFailed: t("channel.saveFailed"),
        deleteFailed: t("channel.deleteFailed"),
        created: (payload) => t("channel.created", { name: payload.name }),
        updated: (_item, payload) => t("channel.updated", { name: payload.name }),
        deleted: (item) => t("channel.deleted", { name: item.name }),
      }}
    />
  )
}
