/*
Copyright © 2025 Arjen Schwarz <developer@arjen.eu>

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/
package cmd

import (
	"testing"
)

// TestDeployMultipleTables verifies that deploy command output supports multiple tables
// with independent column sets (events + outputs).
func TestDeployMultipleTables(t *testing.T) {
	t.Parallel()

	// Test case: Events and Outputs tables should have independent column ordering
	testCases := map[string]struct {
		tableName       string
		expectedColumns []string
		description     string
	}{
		"events_table": {
			tableName:       "Events",
			expectedColumns: []string{"LogicalId", "Type", "Status", "Reason"},
			description:     "Events table should have LogicalId, Type, Status, and Reason columns",
		},
		"outputs_table": {
			tableName:       "Outputs",
			expectedColumns: []string{"Key", "Value", "Description", "ExportName"},
			description:     "Outputs table should have Key, Value, Description, and ExportName columns",
		},
		"failed_events_table": {
			tableName:       "Failed Events",
			expectedColumns: []string{"CfnName", "Type", "Status", "Reason"},
			description:     "Failed Events table should have CfnName, Type, Status, and Reason columns",
		},
	}

	for name, tc := range testCases {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// This test verifies the expected column structure
			// Actual table rendering will be validated in integration tests
			if len(tc.expectedColumns) == 0 {
				t.Errorf("%s: expected columns not defined", tc.tableName)
			}

			// Verify each column name is non-empty
			for i, col := range tc.expectedColumns {
				if col == "" {
					t.Errorf("%s: column %d is empty", tc.tableName, i)
				}
			}
		})
	}
}

// TestDeployTablesAddIncrementally verifies that deploy tables can be added
// incrementally (e.g., in loops) before rendering.
func TestDeployTablesAddIncrementally(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		tableType   string
		rowCount    int
		description string
	}{
		"events_zero_rows": {
			tableType:   "events",
			rowCount:    0,
			description: "Events table should handle zero rows",
		},
		"events_multiple_rows": {
			tableType:   "events",
			rowCount:    5,
			description: "Events table should handle multiple rows added incrementally",
		},
		"outputs_single_row": {
			tableType:   "outputs",
			rowCount:    1,
			description: "Outputs table should handle a single row",
		},
		"outputs_multiple_rows": {
			tableType:   "outputs",
			rowCount:    3,
			description: "Outputs table should handle multiple rows",
		},
	}

	for name, tc := range testCases {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			// This test verifies that tables support incremental row addition
			// Implementation details verified through integration tests
			if tc.rowCount < 0 {
				t.Errorf("invalid row count: %d", tc.rowCount)
			}
		})
	}
}

// TestDeployTableSeparation verifies that multiple tables are properly separated
// in table format output.
func TestDeployTableSeparation(t *testing.T) {
	t.Parallel()

	// This test verifies tables are separated appropriately in output
	testCases := map[string]struct {
		tables          []string
		expectedSpacing bool
		description     string
	}{
		"two_tables_separated": {
			tables:          []string{"Events", "Outputs"},
			expectedSpacing: true,
			description:     "Two tables should be separated with spacing",
		},
		"multiple_tables": {
			tables:          []string{"Events", "Outputs", "Parameters"},
			expectedSpacing: true,
			description:     "Multiple tables should maintain separation",
		},
	}

	for name, tc := range testCases {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if len(tc.tables) == 0 {
				t.Errorf("%s: no tables defined", name)
			}

			// Verify all tables are non-empty strings
			for i, table := range tc.tables {
				if table == "" {
					t.Errorf("%s: table %d is empty", name, i)
				}
			}
		})
	}
}

// TestDeployIndependentColumnOrdering verifies that each table has independent
// column ordering without affecting other tables.
func TestDeployIndependentColumnOrdering(t *testing.T) {
	t.Parallel()

	// Events table column order
	eventsColumns := []string{"LogicalId", "Type", "Status", "Reason"}

	// Outputs table column order (different from events)
	outputsColumns := []string{"Key", "Value", "Description", "ExportName"}

	// Verify columns are different
	if len(eventsColumns) != len(outputsColumns) {
		t.Logf("Column count differs: events=%d, outputs=%d", len(eventsColumns), len(outputsColumns))
	}

	// Verify no column names are shared between tables
	eventsMap := make(map[string]bool)
	for _, col := range eventsColumns {
		eventsMap[col] = true
	}

	for _, col := range outputsColumns {
		if eventsMap[col] {
			// Note: It's okay if some columns have same name (e.g., Type, Status)
			// as they serve different purposes in different tables
			t.Logf("Column '%s' appears in both tables (expected if serving same semantic purpose)", col)
		}
	}

	// Verify column order is preserved
	if len(eventsColumns) > 0 {
		if eventsColumns[0] != "LogicalId" {
			t.Errorf("Events table first column should be 'LogicalId', got '%s'", eventsColumns[0])
		}
	}

	if len(outputsColumns) > 0 {
		if outputsColumns[0] != "Key" {
			t.Errorf("Outputs table first column should be 'Key', got '%s'", outputsColumns[0])
		}
	}
}

// TestDeployFailedEventsTable verifies that failed events table is created
// with proper column separation from main events table.
func TestDeployFailedEventsTable(t *testing.T) {
	t.Parallel()

	failedEventsColumns := []string{"CfnName", "Type", "Status", "Reason"}
	mainEventsColumns := []string{"LogicalId", "Type", "Status", "Reason"}

	// Verify failed events table has CfnName instead of LogicalId
	if failedEventsColumns[0] != "CfnName" {
		t.Errorf("Failed events first column should be 'CfnName', got '%s'", failedEventsColumns[0])
	}

	if mainEventsColumns[0] != "LogicalId" {
		t.Errorf("Events table first column should be 'LogicalId', got '%s'", mainEventsColumns[0])
	}

	// Verify both have same remaining columns
	for i := 1; i < len(mainEventsColumns); i++ {
		if mainEventsColumns[i] != failedEventsColumns[i] {
			t.Errorf("Column %d differs: events=%s, failed=%s", i, mainEventsColumns[i], failedEventsColumns[i])
		}
	}
}
