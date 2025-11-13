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

async function getData(): Promise<Payment[]> {
  // Fetch data from your API here.
  return [
    {
      id: '728ed52f',
      amount: 100,
      status: 'pending',
      email: 'm@example.com',
    },
    // ...
  ];
}

async function RouteComponent() {
  const data = await getData();

  return (
    <>
      <CraftingControls />
      <DataTable columns={columns} data={data} />
    </>
  );
}
