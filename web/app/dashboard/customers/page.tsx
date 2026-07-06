"use client";

import { BanIcon, CheckCircle2Icon } from "lucide-react";
import { useEffect, useMemo, useState } from "react";
import { useSearchParams } from "next/navigation";

import { type CustomerFormSavePayload } from "@/components/customer-form";
import {
  DashboardCrudPage,
  createDashboardStatusColumn,
  createDashboardStatusToggleAction,
  type DashboardCrudColumn,
  type DashboardCrudFilter,
} from "@/components/dashboard/crud";
import { type ComboboxOption } from "@/components/option-combobox";
import { fetchCompanies, type AdminCompany } from "@/lib/api/company";
import {
  deleteCustomer,
  fetchCustomers,
  saveCustomerProfile,
  updateCustomerStatus,
  type AdminCustomer,
} from "@/lib/api/customer";
import { Gender, Status } from "@/lib/generated/enums";
import { useI18n } from "@/i18n/provider";
import { EditDialog } from "./_components/edit";

type TFunction = (key: string, values?: Record<string, string | number>) => string;

function getGenderText(gender: number, t: TFunction) {
  if (gender === Gender.Male) return t("customerForm.genderMale");
  if (gender === Gender.Female) return t("customerForm.genderFemale");
  return t("customerForm.genderUnknown");
}

export default function DashboardCustomersPage() {
  const t = useI18n();
  const searchParams = useSearchParams();
  const initialKeyword = searchParams.get("keyword") ?? "";
  const [companyOptions, setCompanyOptions] = useState<ComboboxOption[]>([
    { value: "0", label: t("customer.allCompanies") },
  ]);
  const [companyNameMap, setCompanyNameMap] = useState<Record<number, string>>(
    {},
  );

  const listStatusOptions = useMemo(
    () => [
      { value: "all", label: t("status.all") },
      { value: String(Status.Ok), label: t("status.ok") },
      { value: String(Status.Disabled), label: t("status.disabled") },
    ],
    [t],
  );
  const genderOptions = useMemo(
    () => [
      { value: "all", label: t("customer.allGenders") },
      { value: String(Gender.Unknown), label: t("customerForm.genderUnknown") },
      { value: String(Gender.Male), label: t("customerForm.genderMale") },
      { value: String(Gender.Female), label: t("customerForm.genderFemale") },
    ],
    [t],
  );

  useEffect(() => {
    async function loadCompanies() {
      try {
        const data = await fetchCompanies({ status: 0, page: 1, limit: 500 });
        setCompanyOptions([
          { value: "0", label: t("customer.allCompanies") },
          ...data.results.map((item) => ({
            value: String(item.id),
            label: item.name,
          })),
        ]);
        const map: Record<number, string> = {};
        data.results.forEach((item: AdminCompany) => {
          map[item.id] = item.name;
        });
        setCompanyNameMap(map);
      } catch {
        // Company names are optional display enrichment for this list.
      }
    }
    void loadCompanies();
  }, [t]);

  const filters = useMemo<DashboardCrudFilter[]>(
    () => [
      {
        name: "keyword",
        label: t("customer.columnName"),
        placeholder: t("customer.keywordPlaceholder"),
        defaultValue: initialKeyword,
        trim: true,
        className: "w-full sm:w-72",
      },
      {
        name: "gender",
        label: t("customer.columnGender"),
        type: "select",
        defaultValue: "all",
        allValue: "all",
        valueType: "number",
        options: genderOptions,
        className: "w-full sm:w-36",
      },
      {
        name: "companyId",
        label: t("customer.columnCompany"),
        type: "select",
        defaultValue: "0",
        allValue: "0",
        valueType: "number",
        options: companyOptions,
        className: "w-full sm:w-56",
      },
      {
        name: "status",
        label: t("customer.columnStatus"),
        type: "select",
        defaultValue: "all",
        allValue: "all",
        valueType: "number",
        options: listStatusOptions,
        className: "w-full sm:w-36",
      },
    ],
    [companyOptions, genderOptions, initialKeyword, listStatusOptions, t],
  );

  const columns = useMemo<DashboardCrudColumn<AdminCustomer>[]>(
    () => [
      {
        key: "id",
        label: "ID",
        className: "w-20",
        render: (item) => item.id,
      },
      {
        key: "name",
        label: t("customer.columnName"),
        render: (item) => <span className="font-medium">{item.name}</span>,
      },
      {
        key: "gender",
        label: t("customer.columnGender"),
        className: "w-20",
        render: (item) => (
          <span className="text-muted-foreground">
            {getGenderText(item.gender, t)}
          </span>
        ),
      },
      {
        key: "company",
        label: t("customer.columnCompany"),
        render: (item) => (
          <span className="text-muted-foreground">
            {item.companyId > 0
              ? (companyNameMap[item.companyId] ?? String(item.companyId))
              : "-"}
          </span>
        ),
      },
      {
        key: "mobile",
        label: t("customer.columnMobile"),
        render: (item) => (
          <span className="text-muted-foreground">
            {item.primaryMobile || "-"}
          </span>
        ),
      },
      {
        key: "email",
        label: t("customer.columnEmail"),
        render: (item) => (
          <span className="text-muted-foreground">
            {item.primaryEmail || "-"}
          </span>
        ),
      },
      createDashboardStatusColumn<AdminCustomer, number>({
        label: t("customer.columnStatus"),
        className: "w-24",
        getStatus: (item) => item.status,
        getLabel: (status) =>
          status === Status.Ok ? t("status.ok") : t("status.disabled"),
        getBadgeVariant: (status) =>
          status === Status.Ok ? "default" : "secondary",
      }),
    ],
    [companyNameMap, t],
  );

  return (
    <DashboardCrudPage<AdminCustomer, CustomerFormSavePayload>
      filters={filters}
      columns={columns}
      fetchList={(query) =>
        fetchCustomers({
          keyword:
            typeof query.keyword === "string" ? query.keyword : undefined,
          status:
            typeof query.status === "number" ? query.status : undefined,
          gender:
            typeof query.gender === "number" ? query.gender : undefined,
          companyId:
            typeof query.companyId === "number" ? query.companyId : undefined,
          page: Number(query.page),
          limit: Number(query.limit),
        })
      }
      getItemId={(item) => item.id}
      createItem={saveCustomerProfile}
      updateItem={(_item, payload) => saveCustomerProfile(payload)}
      deleteItem={(item) => deleteCustomer(item.id)}
      canDelete={(item) => item.status !== Status.Deleted}
      rowActions={[
        createDashboardStatusToggleAction<AdminCustomer, number>({
          icon: (item) =>
            item.status === Status.Ok ? <BanIcon /> : <CheckCircle2Icon />,
          label: (item) =>
            item.status === Status.Ok
              ? t("customer.disable")
              : t("customer.enable"),
          disabled: (item) => item.status === Status.Deleted,
          getNextStatus: (item) =>
            item.status === Status.Ok ? Status.Disabled : Status.Ok,
          updateStatus: (item, nextStatus) =>
            updateCustomerStatus(item.id, nextStatus),
          successMessage: (item, nextStatus) =>
            t(nextStatus === Status.Ok ? "customer.enabled" : "customer.disabled", {
              name: item.name,
            }),
          errorMessage: t("customer.statusUpdateFailed"),
        }),
      ]}
      renderEditDialog={({ open, saving, itemId, onOpenChange, onSubmit }) => (
        <EditDialog
          open={open}
          saving={saving}
          itemId={itemId}
          onOpenChange={onOpenChange}
          onSave={onSubmit}
        />
      )}
      labels={{
        refresh: t("customer.refresh"),
        create: t("customer.new"),
        query: t("customer.query"),
        loading: t("customer.loading"),
        empty: t("customer.empty"),
        actions: t("customer.columnActions"),
        edit: t("customer.edit"),
        delete: t("customer.delete"),
        processing: t("customer.processing"),
        moreActions: (item) => t("customer.moreActions", { name: item.name }),
        loadFailed: t("customer.loadFailed"),
        saveFailed: t("customer.saveFailed"),
        deleteFailed: t("customer.deleteFailed"),
        created: (payload) => t("customer.created", { name: payload.name }),
        updated: (item) => t("customer.updated", { name: item.name }),
        deleted: (item) => t("customer.deleted", { name: item.name }),
      }}
    />
  );
}
