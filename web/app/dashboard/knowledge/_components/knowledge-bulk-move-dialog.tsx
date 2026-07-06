"use client";

import { useEffect, useMemo, useState } from "react";

import { OptionCombobox } from "@/components/option-combobox";
import { ProjectDialog } from "@/components/project-dialog";
import { Button } from "@/components/ui/button";
import {
  fetchKnowledgeDirectories,
  type KnowledgeDirectory,
} from "@/lib/api/admin";
import { useI18n } from "@/i18n/provider";

type KnowledgeBulkMoveDialogProps = {
  open: boolean;
  knowledgeBaseId: number;
  moving: boolean;
  selectedCount: number;
  onOpenChange: (open: boolean) => void;
  onSubmit: (directoryId: number) => void;
};

function flattenDirectoryOptions(
  directories: KnowledgeDirectory[],
): Array<{ value: string; label: string }> {
  const result: Array<{ value: string; label: string }> = [];
  for (const item of directories) {
    result.push({ value: String(item.id), label: item.name });
    for (const child of item.children ?? []) {
      result.push({ value: String(child.id), label: `${item.name} / ${child.name}` });
    }
  }
  return result;
}

export function KnowledgeBulkMoveDialog({
  open,
  knowledgeBaseId,
  moving,
  selectedCount,
  onOpenChange,
  onSubmit,
}: KnowledgeBulkMoveDialogProps) {
  const t = useI18n();
  const [directories, setDirectories] = useState<KnowledgeDirectory[]>([]);
  const [targetDirectoryId, setTargetDirectoryId] = useState("0");

  useEffect(() => {
    if (!open) {
      return;
    }
    const resetTask = window.setTimeout(() => {
      setTargetDirectoryId("0");
    }, 0);
    fetchKnowledgeDirectories(knowledgeBaseId)
      .then(setDirectories)
      .catch((error) => {
        console.error(error);
        setDirectories([]);
      });
    return () => window.clearTimeout(resetTask);
  }, [knowledgeBaseId, open]);

  const directoryOptions = useMemo(() => [
    { value: "0", label: t("knowledge.rootContent") },
    ...flattenDirectoryOptions(directories),
  ], [directories, t]);

  return (
    <ProjectDialog
      open={open}
      onOpenChange={onOpenChange}
      title={t("knowledge.batchMove")}
      description={t("knowledge.batchMoveDescription", { count: selectedCount })}
      size="sm"
      footer={
        <>
          <Button
            type="button"
            variant="outline"
            onClick={() => onOpenChange(false)}
            disabled={moving}
          >
            {t("knowledge.cancel")}
          </Button>
          <Button
            type="button"
            onClick={() => onSubmit(Number(targetDirectoryId))}
            disabled={moving || selectedCount <= 0}
          >
            {moving ? t("knowledge.moving") : t("knowledge.move")}
          </Button>
        </>
      }
    >
      <div className="space-y-2">
        <div className="text-sm font-medium">{t("knowledge.targetDirectory")}</div>
        <OptionCombobox
          value={targetDirectoryId}
          onChange={(value) => setTargetDirectoryId(value ?? "0")}
          options={directoryOptions}
          placeholder={t("knowledge.selectDirectory")}
          searchPlaceholder={t("knowledge.searchDirectory")}
          emptyText={t("knowledge.emptyDirectory")}
        />
      </div>
    </ProjectDialog>
  );
}
