import { Card } from "./card";

export function LoadingState({ label = "Loading..." }: { label?: string }) {
  return <Card className="text-sm text-slate-500">{label}</Card>;
}

export function EmptyState({
  title,
  body
}: {
  title: string;
  body: string;
}) {
  return (
    <Card>
      <h3 className="text-lg font-semibold text-slate-900">{title}</h3>
      <p className="mt-2 text-sm text-slate-600">{body}</p>
    </Card>
  );
}

export function ErrorState({
  title,
  body
}: {
  title: string;
  body: string;
}) {
  return (
    <Card className="border-rose-200 bg-rose-50/90">
      <h3 className="text-lg font-semibold text-rose-900">{title}</h3>
      <p className="mt-2 text-sm text-rose-700">{body}</p>
    </Card>
  );
}
