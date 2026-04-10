"use client";

import { useParams } from "next/navigation";
import { ScheduledSessionForm } from "../_components/scheduled-session-form";

export default function CreateScheduledSessionPage() {
  const params = useParams<{ name: string }>();
  const projectName = params.name;

  return <ScheduledSessionForm projectName={projectName} mode="create" />;
}
