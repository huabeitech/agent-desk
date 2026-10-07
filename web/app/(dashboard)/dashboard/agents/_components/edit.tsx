"use client";

import { zodResolver } from "@hookform/resolvers/zod";
import { CheckIcon, ChevronsUpDownIcon } from "lucide-react";
import { useCallback, useEffect, useMemo, useState } from "react";
import { Controller, type Resolver, useForm } from "react-hook-form";
import { toast } from "sonner";
import { z } from "zod/v4";

import { ImageInput } from "@/components/image-input";
import { OptionCombobox } from "@/components/option-combobox";
import { ProjectDialog } from "@/components/project-dialog";
import { Button } from "@/components/ui/button";
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
} from "@/components/ui/command";
import {
  Field,
  FieldContent,
  FieldError,
  FieldLabel,
} from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import { Switch } from "@/components/ui/switch";
import { Textarea } from "@/components/ui/textarea";
import {
  fetchAgentProfile,
  fetchUsersAll,
  type AdminAgentProfile,
  type AdminUser,
  type CreateAdminAgentProfilePayload,
} from "@/lib/api/admin";
import { useI18n } from "@/i18n/provider";
import { ServiceStatus } from "@/lib/generated/enums";

type TFunction = (key: string, values?: Record<string, string | number>) => string;

type AgentEditDialogProps = {
  open: boolean;
  saving: boolean;
  itemId: number | null;
  defaultTeamId: number | null;
  onOpenChange: (open: boolean) => void;
  onSubmit: (payload: CreateAdminAgentProfilePayload) => Promise<void>;
};

const emptyForm: EditForm = {
  userId: "",
  teamId: "",
  agentCode: "",
  displayName: "",
  avatar: "",
  serviceStatus: String(ServiceStatus.Idle) as "0" | "1",
  maxConcurrentCount: "0",
  priorityLevel: "0",
  autoAssignEnabled: true,
  receiveOfflineMessage: false,
  remark: "",
};

type EditForm = {
  userId: string;
  teamId: string;
  agentCode: string;
  displayName: string;
  avatar: string;
  serviceStatus: "0" | "1";
  maxConcurrentCount: string;
  priorityLevel: string;
  autoAssignEnabled: boolean;
  receiveOfflineMessage: boolean;
  remark: string;
};

function createEditFormSchema(t: TFunction) {
  return z.object({
  userId: z.string().trim().min(1, t("agentProfile.userRequired")),
  teamId: z.string().trim().min(1, t("agentProfile.teamRequired")),
  agentCode: z.string().trim().min(1, t("agentProfile.agentCodeRequired")),
  displayName: z.string().trim().min(1, t("agentProfile.displayNameRequired")),
  avatar: z.string().trim(),
  serviceStatus: z.enum(["0", "1"], {
    message: t("agentProfile.statusRequired"),
  }),
  maxConcurrentCount: z
    .string()
    .trim()
    .regex(/^\d+$/, t("agentProfile.maxConcurrentInvalid")),
  priorityLevel: z
    .string()
    .trim()
    .regex(/^-?\d+$/, t("agentProfile.priorityInvalid")),
  autoAssignEnabled: z.boolean(),
  receiveOfflineMessage: z.boolean(),
  remark: z.string().trim(),
  });
}

function getServiceStatusOptions(t: TFunction) {
  return [
    { value: String(ServiceStatus.Idle), label: t("agentProfile.statusIdle") },
    { value: String(ServiceStatus.Busy), label: t("agentProfile.statusBusy") },
  ];
}

function buildForm(item: AdminAgentProfile | null): EditForm {
  if (!item) {
    return emptyForm;
  }
  return {
    userId: String(item.userId),
    teamId: String(item.teamId),
    agentCode: item.agentCode,
    displayName: item.displayName,
    avatar: item.avatarAssetId || item.avatar || "",
    serviceStatus: String(item.serviceStatus) as EditForm["serviceStatus"],
    maxConcurrentCount: String(item.maxConcurrentCount),
    priorityLevel: String(item.priorityLevel),
    autoAssignEnabled: item.autoAssignEnabled,
    receiveOfflineMessage: item.receiveOfflineMessage,
    remark: item.remark || "",
  };
}

function buildFormWithDefaultTeam(
  item: AdminAgentProfile | null,
  defaultTeamId: number | null,
): EditForm {
  const form = buildForm(item);
  if (!item && defaultTeamId) {
    return {
      ...form,
      teamId: String(defaultTeamId),
    };
  }
  return form;
}

function buildPayload(form: EditForm): CreateAdminAgentProfilePayload {
  return {
    userId: Number(form.userId),
    teamId: Number(form.teamId),
    agentCode: form.agentCode.trim(),
    displayName: form.displayName.trim(),
    avatar: form.avatar.trim(),
    serviceStatus: Number(form.serviceStatus),
    maxConcurrentCount: Number(form.maxConcurrentCount),
    priorityLevel: Number(form.priorityLevel),
    autoAssignEnabled: form.autoAssignEnabled,
    receiveOfflineMessage: form.receiveOfflineMessage,
    remark: form.remark.trim(),
  };
}

export function EditDialog({
  open,
  saving,
  itemId,
  defaultTeamId,
  onOpenChange,
  onSubmit,
}: AgentEditDialogProps) {
  if (!open) {
    return null;
  }

  return (
    <AgentEditDialogBody
      key={itemId ? `edit-${itemId}` : "create"}
      open={open}
      itemId={itemId}
      defaultTeamId={defaultTeamId}
      saving={saving}
      onOpenChange={onOpenChange}
      onSubmit={onSubmit}
    />
  );
}

type AgentEditDialogBodyProps = {
  open: boolean;
  saving: boolean;
  itemId: number | null;
  defaultTeamId: number | null;
  onOpenChange: (open: boolean) => void;
  onSubmit: (payload: CreateAdminAgentProfilePayload) => Promise<void>;
};

function AgentEditDialogBody({
  open,
  saving,
  itemId,
  defaultTeamId,
  onOpenChange,
  onSubmit,
}: AgentEditDialogBodyProps) {
  const t = useI18n();
  const [users, setUsers] = useState<AdminUser[]>([]);
  const [userSelectOpen, setUserSelectOpen] = useState(false);
  const [loading, setLoading] = useState(false);
  const [avatarPreview, setAvatarPreview] = useState("");
  const userOptions = users.map((user) => ({
    value: String(user.id),
    label: `${user.nickname || user.username} (${user.username})`,
  }));
  const serviceStatusOptions = useMemo(() => getServiceStatusOptions(t), [t]);
  const loadOptions = useCallback(async () => {
    try {
      const [usersData] = await Promise.all([
        fetchUsersAll(),
      ]);
      setUsers(usersData);
    } catch (error) {
      toast.error(error instanceof Error ? error.message : t("agentProfile.loadOptionsFailed"));
    }
  }, [t]);
  const editFormSchema = useMemo(() => createEditFormSchema(t), [t]);
  const editFormResolver = useMemo(
    () => zodResolver(editFormSchema) as Resolver<EditForm>,
    [editFormSchema],
  );
  const form = useForm<EditForm>({
    resolver: editFormResolver,
    defaultValues: buildFormWithDefaultTeam(null, defaultTeamId),
  });
  const {
    control,
    handleSubmit,
    reset,
    register,
    formState: { errors },
  } = form;

  useEffect(() => {
    async function loadDetail() {
      if (!itemId) {
        setAvatarPreview("");
        reset(buildFormWithDefaultTeam(null, defaultTeamId));
        return;
      }
      setLoading(true);
      try {
        const data = await fetchAgentProfile(itemId);
        setAvatarPreview(data.avatar || "");
        reset(buildForm(data));
      } catch (error) {
        toast.error(error instanceof Error ? error.message : t("agentProfile.loadDetailFailed"));
      } finally {
        setLoading(false);
      }
    }
    void loadDetail();
  }, [itemId, defaultTeamId, reset, t]);

  useEffect(() => {
    if (open) {
      void loadOptions();
    }
  }, [loadOptions, open]);

  async function onFormSubmit(values: EditForm) {
    await onSubmit(buildPayload(values));
  }

  const formId = "agent-edit-form";

  return (
    <ProjectDialog
      open={open}
      onOpenChange={onOpenChange}
      title={itemId ? t("agentProfile.editTitle") : t("agentProfile.createTitle")}
      size="lg"
      footer={
        <>
          <Button
            type="button"
            variant="outline"
            onClick={() => onOpenChange(false)}
            disabled={saving}
          >
            {t("agentProfile.cancel")}
          </Button>
          <Button type="submit" form={formId} disabled={saving || loading}>
            {saving ? t("agentProfile.saving") : t("agentProfile.save")}
          </Button>
        </>
      }
    >
      {loading ? (
        <div className="flex items-center justify-center py-12">
          <div className="text-muted-foreground">{t("agentProfile.loading")}</div>
        </div>
      ) : (
        <form
          id={formId}
          onSubmit={handleSubmit(onFormSubmit)}
          className="space-y-4"
        >
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
            <Field data-invalid={!!errors.userId}>
              <FieldLabel>{t("agentProfile.linkedUser")}</FieldLabel>
              <FieldContent>
                <Controller
                  control={control}
                  name="userId"
                  render={({ field }) => (
                    <Popover
                      open={userSelectOpen}
                      onOpenChange={setUserSelectOpen}
                    >
                      <PopoverTrigger
                        render={
                          <Button
                            variant="outline"
                            role="combobox"
                            aria-expanded={userSelectOpen}
                            className="w-full justify-between font-normal"
                          />
                        }
                      >
                        <span className="truncate">
                          {userOptions.find(
                            (option) => option.value === field.value,
                          )?.label ?? t("agentProfile.selectUser")}
                        </span>
                        <ChevronsUpDownIcon className="ml-2 size-4 shrink-0 opacity-50" />
                      </PopoverTrigger>
                      <PopoverContent
                        className="w-[var(--radix-popper-anchor-width)] p-0"
                        align="start"
                      >
                        <Command>
                          <CommandInput placeholder={t("agentProfile.searchUser")} />
                          <CommandList>
                            <CommandEmpty>{t("agentProfile.emptyUser")}</CommandEmpty>
                            <CommandGroup>
                              {userOptions.map((option) => (
                                <CommandItem
                                  key={option.value}
                                  value={option.label}
                                  onSelect={() => {
                                    field.onChange(option.value);
                                    setUserSelectOpen(false);
                                  }}
                                >
                                  <CheckIcon
                                    className={`mr-2 size-4 ${
                                      field.value === option.value
                                        ? "opacity-100"
                                        : "opacity-0"
                                    }`}
                                  />
                                  {option.label}
                                </CommandItem>
                              ))}
                            </CommandGroup>
                          </CommandList>
                        </Command>
                      </PopoverContent>
                    </Popover>
                  )}
                />
                <FieldError errors={[errors.userId]} />
              </FieldContent>
            </Field>
            <Field data-invalid={!!errors.displayName}>
              <FieldLabel htmlFor="agent-display-name">{t("agentProfile.displayName")}</FieldLabel>
              <FieldContent>
                <Input
                  id="agent-display-name"
                  placeholder={t("agentProfile.displayNamePlaceholder")}
                  {...register("displayName")}
                />
                <FieldError errors={[errors.displayName]} />
              </FieldContent>
            </Field>
          </div>

          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
            <Field data-invalid={!!errors.agentCode}>
              <FieldLabel htmlFor="agent-code">{t("agentProfile.agentCodeLabel")}</FieldLabel>
              <FieldContent>
                <Input
                  id="agent-code"
                  placeholder={t("agentProfile.agentCodePlaceholder")}
                  {...register("agentCode")}
                />
                <FieldError errors={[errors.agentCode]} />
              </FieldContent>
            </Field>

            <Field className="min-h-32">
              <FieldLabel>{t("agentProfile.avatar")}</FieldLabel>
              <FieldContent>
                <Controller
                  control={control}
                  name="avatar"
                  render={({ field }) => (
                    <ImageInput
                      value={field.value}
                      previewValue={avatarPreview}
                      onChange={field.onChange}
                      disabled={saving}
                      prefix="avatar"
                      placeholder={t("agentProfile.avatarUpload")}
                      className="size-16 rounded-full"
                    />
                  )}
                />
              </FieldContent>
            </Field>
          </div>

          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
            <Field data-invalid={!!errors.serviceStatus}>
              <FieldLabel>{t("agentProfile.serviceStatus")}</FieldLabel>
              <FieldContent>
                <Controller
                  control={control}
                  name="serviceStatus"
                  render={({ field }) => (
                    <OptionCombobox
                      options={serviceStatusOptions}
                      value={field.value}
                      onChange={field.onChange}
                      placeholder={t("agentProfile.selectStatus")}
                      searchPlaceholder={t("agentProfile.searchStatus")}
                      emptyText={t("agentProfile.emptyStatus")}
                    />
                  )}
                />
                <FieldError errors={[errors.serviceStatus]} />
              </FieldContent>
            </Field>
          </div>

          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
            <Field data-invalid={!!errors.maxConcurrentCount}>
              <FieldLabel htmlFor="agent-max-concurrent-count">
                {t("agentProfile.maxConcurrent")}
              </FieldLabel>
              <FieldContent>
                <Input
                  id="agent-max-concurrent-count"
                  type="number"
                  min={0}
                  {...register("maxConcurrentCount")}
                />
                <FieldError errors={[errors.maxConcurrentCount]} />
              </FieldContent>
            </Field>
            <Field data-invalid={!!errors.priorityLevel}>
              <FieldLabel htmlFor="agent-priority-level">{t("agentProfile.priority")}</FieldLabel>
              <FieldContent>
                <Input
                  id="agent-priority-level"
                  type="number"
                  step={1}
                  {...register("priorityLevel")}
                />
                <FieldError errors={[errors.priorityLevel]} />
              </FieldContent>
            </Field>
          </div>

          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
            <Field>
              <FieldLabel>{t("agentProfile.autoAssignEnabled")}</FieldLabel>
              <FieldContent>
                <Controller
                  control={control}
                  name="autoAssignEnabled"
                  render={({ field }) => (
                    <Switch
                      checked={field.value}
                      onCheckedChange={field.onChange}
                    />
                  )}
                />
              </FieldContent>
            </Field>
            <Field>
              <FieldLabel>{t("agentProfile.receiveOfflineMessage")}</FieldLabel>
              <FieldContent>
                <Controller
                  control={control}
                  name="receiveOfflineMessage"
                  render={({ field }) => (
                    <Switch
                      checked={field.value}
                      onCheckedChange={field.onChange}
                    />
                  )}
                />
              </FieldContent>
            </Field>
          </div>

          <Field>
            <FieldLabel htmlFor="agent-remark">{t("agentProfile.remark")}</FieldLabel>
            <FieldContent>
              <Textarea
                id="agent-remark"
                rows={4}
                placeholder={t("agentProfile.remarkPlaceholder")}
                {...register("remark")}
              />
            </FieldContent>
          </Field>
        </form>
      )}
    </ProjectDialog>
  );
}
