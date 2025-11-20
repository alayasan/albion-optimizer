import { CraftingControls } from '@/components/crafting/controls';
import { DataTable } from '@/components/ui/data-table';
import { createFileRoute } from '@tanstack/react-router';
// import { ColumnDef } from '@tanstack/react-table';
import type { Payment } from 'shared/dist';

export const Route = createFileRoute('/crafting')({
  component: RouteComponent,
});

const columns = [
  {
    accessorKey: 'status',
    header: 'Status',
  },
  {
    accessorKey: 'email',
    header: 'Email',
  },
  {
    accessorKey: 'amount',
    header: 'Amount',
  },
];

function RouteComponent() {
  // TODO: Use React Query or similar for async data fetching
  // For now, using dummy data synchronously
  const data: Payment[] = [
    {
      id: '728ed52f',
      amount: 100,
      status: 'pending',
      email: 'm@example.com',
    },
  ];

  return (
    <>
      <CraftingControls />
      <DataTable columns={columns} data={data} />
    </>
  );
}
