import type { Message } from "../../api";

export function HeadersPanel({ message }: { message: Message }) {
  const rows = Object.entries(message.headers)
    .sort(([a], [b]) => a.localeCompare(b))
    .flatMap(([name, values]) => values.map((value, i) => ({ key: `${name}-${i}`, name, value })));
  return (
    <table className="w-full text-[13px]">
      <tbody>
        {rows.map((r) => (
          <tr key={r.key} className="border-b border-zinc-100 align-top dark:border-zinc-900">
            <th
              scope="row"
              className="w-1/4 py-2 pr-4 pl-5 text-left font-medium whitespace-nowrap text-zinc-500"
            >
              {r.name}
            </th>
            <td className="py-2 pr-5 font-mono break-all">{r.value}</td>
          </tr>
        ))}
      </tbody>
    </table>
  );
}
