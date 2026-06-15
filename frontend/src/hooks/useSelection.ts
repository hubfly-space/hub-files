import { useCallback, useState } from "react";

export function useSelection() {
  const [selectedItems, setSelectedItems] = useState<Set<string>>(new Set());
  const [selectionMode, setSelectionMode] = useState(false);

  const clearSelection = useCallback(() => {
    setSelectedItems(new Set());
  }, []);

  const toggleSelectionMode = useCallback(() => {
    setSelectionMode((currentMode) => {
      if (currentMode) {
        setSelectedItems(new Set());
      }

      return !currentMode;
    });
  }, []);

  const handleSelect = useCallback((name: string, multi: boolean) => {
    setSelectedItems((previousItems) => {
      const nextItems = new Set(previousItems);

      if (multi) {
        if (nextItems.has(name)) nextItems.delete(name);
        else nextItems.add(name);

        return nextItems;
      }

      if (nextItems.has(name) && nextItems.size === 1) {
        nextItems.delete(name);
      } else {
        nextItems.clear();
        nextItems.add(name);
      }

      return nextItems;
    });
  }, []);

  return {
    selectedItems,
    selectionMode,
    setSelectedItems,
    clearSelection,
    toggleSelectionMode,
    handleSelect,
  };
}
