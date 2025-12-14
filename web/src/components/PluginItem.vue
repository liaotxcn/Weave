<template>
  <el-card
    @click="handleClick"
    :class="['plugin-item', { active: isActive }]"
    :body-style="{ padding: '12px' }"
    :bordered="isActive"
    shadow="hover"
  >
    <div class="plugin-content">
      <div class="item-header">
        <span class="item-title">{{ name }}</span>
        <el-badge
          v-if="badgeCount !== undefined"
          :value="badgeCount"
          :max="99"
          class="item-badge"
          type="primary"
        ></el-badge>
      </div>
      <span v-if="description" class="item-desc">{{ description }}</span>
    </div>
  </el-card>
</template>

<script setup>
const props = defineProps({
  name: {
    type: String,
    required: true
  },
  description: String,
  badgeCount: Number,
  isActive: {
    type: Boolean,
    default: false
  }
});

const emit = defineEmits(['select']);

const handleClick = () => {
  emit('select', props.name);
};
</script>

<style scoped>
.plugin-item {
  cursor: pointer;
  transition: all 0.3s ease;
  margin-bottom: var(--space-2);
  border-radius: 8px;
}

.plugin-item:hover {
  transform: translateY(-2px);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
}

.plugin-item.active {
  border-color: var(--color-primary);
  background-color: rgba(99, 102, 241, 0.05);
}

.plugin-content {
  display: flex;
  align-items: center;
  width: 100%;
  gap: var(--space-3);
}

.item-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 40px;
  height: 40px;
  border-radius: 8px;
  background-color: rgba(99, 102, 241, 0.1);
  color: var(--color-primary);
  flex-shrink: 0;
  transition: all 0.3s ease;
}

.plugin-item:hover .item-icon {
  background-color: rgba(99, 102, 241, 0.15);
  transform: scale(1.05);
}

.plugin-item.active .item-icon {
  background-color: rgba(99, 102, 241, 0.2);
}

.item-info {
  flex: 1;
  min-width: 0;
}

.item-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: var(--space-1);
  width: 100%;
}

.item-title {
  font-weight: 500;
  font-size: var(--font-size-sm);
  color: var(--color-text-primary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  flex: 1;
}

.item-desc {
  font-size: var(--font-size-xs);
  opacity: 0.8;
  line-height: 1.4;
  color: var(--color-text-secondary);
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.item-badge {
  margin-left: var(--space-2);
}

.active-indicator {
  color: var(--color-primary);
  font-size: 18px;
  flex-shrink: 0;
  opacity: 0;
  transition: opacity 0.3s ease;
}

.plugin-item.active .active-indicator {
  opacity: 1;
}

/* 自定义 Element Plus 徽章样式 */
:deep(.el-badge__content.is-fixed) {
  right: 0;
  top: 0;
}

:deep(.el-badge__content--primary) {
  background-color: rgba(99, 102, 241, 0.2);
  color: var(--color-primary);
  border: none;
  font-size: var(--font-size-xs);
  padding: 2px 6px;
  min-width: 18px;
  height: 18px;
  line-height: 18px;
}

.plugin-item.active :deep(.el-badge__content--primary) {
  background-color: var(--color-primary);
  color: white;
}
</style>