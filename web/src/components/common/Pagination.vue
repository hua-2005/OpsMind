<template>
  <n-pagination
    :page="currentPage"
    :page-size="pageSize"
    :item-count="total"
    :page-sizes="[10, 20, 50]"
    show-size-picker
    @update:page="$emit('update:current-page', $event)"
    @update:page-size="$emit('update:page-size', $event)"
  />
</template>

<script setup lang="ts">
import { NPagination } from 'naive-ui'

// TODO(component/Pagination): page-size 改变时应重置 currentPage=1。
// 否则从高页码切到大 pageSize 可能请求到空页。
defineProps<{ total: number; currentPage: number; pageSize: number }>()
defineEmits<{
  (e: 'update:current-page', page: number): void
  (e: 'update:page-size', size: number): void
}>()
</script>
