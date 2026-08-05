/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/

type DocsTableProps = {
  headers: string[]
  rows: Array<{ cells: React.ReactNode[]; key: string }>
  minWidth?: 'sm' | 'md' | 'lg'
}

const minWidthClasses = {
  sm: 'min-w-[480px]',
  md: 'min-w-[620px]',
  lg: 'min-w-[760px]',
}

export function DocsTable(props: DocsTableProps) {
  return (
    <div className='border-border overflow-x-auto rounded-lg border'>
      <table
        className={`w-full text-left text-sm ${minWidthClasses[props.minWidth ?? 'md']}`}
      >
        <thead className='bg-muted/40 text-muted-foreground'>
          <tr>
            {props.headers.map((header) => (
              <th key={header} className='px-4 py-3 font-medium'>
                {header}
              </th>
            ))}
          </tr>
        </thead>
        <tbody className='divide-border divide-y'>
          {props.rows.map((row) => (
            <tr key={row.key}>
              {row.cells.map((cell, columnIndex) => (
                <td
                  key={`${row.key}-${props.headers[columnIndex]}`}
                  className='px-4 py-3 align-top leading-6'
                >
                  {cell}
                </td>
              ))}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}
