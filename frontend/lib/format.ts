export function formatCurrency(amount: number, currency: string) {
  try {
    return new Intl.NumberFormat("en-US", {
      style: "currency",
      currency: currency || "USD",
      maximumFractionDigits: 2
    }).format(amount);
  } catch {
    return `${amount.toFixed(2)} ${currency}`;
  }
}

export function formatDateTime(value?: string | null) {
  if (!value) {
    return "n/a";
  }
  return new Intl.DateTimeFormat("en-US", {
    dateStyle: "medium",
    timeStyle: "short"
  }).format(new Date(value));
}

export function formatMetricKey(key: string) {
  return key.replace(/_/g, " ");
}

export function truncateMiddle(value: string, visible = 6) {
  if (value.length <= visible * 2) {
    return value;
  }
  return `${value.slice(0, visible)}...${value.slice(-visible)}`;
}
