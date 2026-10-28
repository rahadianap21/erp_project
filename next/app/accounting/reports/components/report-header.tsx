import React from "react";
import { PageHeader } from "@/components/page-header";

interface ReportHeaderProps {
  icon: React.ComponentType<{ className?: string }>;
  title: string;
  subtitle: string;
  children?: React.ReactNode;
}

export function ReportHeader({
  icon: Icon,
  title,
  subtitle,
  children,
}: ReportHeaderProps) {
  return (
    <PageHeader
      icon={Icon}
      title={title}
      description={subtitle}
      actions={children}
    />
  );
}
