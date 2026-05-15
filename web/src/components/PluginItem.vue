<template>
  <div
    @click="handleClick"
    :class="['plugin-item', { active: isActive }]"
  >
    <div class="item-content">
      <div class="item-header">
        <span class="item-title">{{ name }}</span>
        <span
          v-if="badgeCount !== undefined"
          class="item-badge"
        >
          {{ badgeCount > 99 ? '99+' : badgeCount }}
        </span>
      </div>
      <span v-if="description" class="item-desc">{{ description }}</span>
    </div>
    <div v-if="isActive" class="active-indicator">
      <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3" stroke-linecap="round" stroke-linejoin="round"><polyline points="20 6 9 17 4 12"></polyline></svg>
    </div>
  </div>
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
  padding: 14px 16px;
  background: white;
  border: 1.5px solid var(--border-light);
  border-radius: 12px;
  transition: all 0.25s cubic-bezier(0.4, 0, 0.2, 1);
  position: relative;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.plugin-item:hover {
  transform: translateY(-2px) translateX(2px);
  box-shadow: 0 6px 20px rgba(99, 102, 241, 0.12);
  border-color: rgba(99, 102, 241, 0.3);
  background: linear-gradient(135deg, #ffffff, rgba(99, 102, 241, 0.02));
}

.plugin-item.active {
  border-color: var(--primary-500);
  background: linear-gradient(135deg, rgba(99, 102, 241, 0.06), rgba(139, 92, 246, 0.04));
  box-shadow: 0 4px 16px rgba(99, 102, 241, 0.15), inset 0 1px 0 rgba(255, 255, 255, 0.5);
}

.item-content {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.item-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 8px;
}

.item-title {
  font-weight: 600;
  font-size: 14px;
  color: var(--color-text-primary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  flex: 1;
  letter-spacing: -0.01em;
}

.item-badge {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 22px;
  height: 22px;
  padding: 0 7px;
  font-size: 11px;
  font-weight: 700;
  color: var(--primary-600);
  background: rgba(99, 102, 241, 0.12);
  border-radius: 10px;
  border: 1px solid rgba(99, 102, 241, 0.2);
  flex-shrink: 0;
  transition: all 0.25s ease;
}

.plugin-item.active .item-badge {
  color: white;
  background: var(--primary-500);
  border-color: var(--primary-500);
  box-shadow: 0 2px 8px rgba(99, 102, 241, 0.3);
}

.item-desc {
  font-size: 12px;
  line-height: 1.5;
  color: var(--color-text-secondary);
  opacity: 0.85;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.active-indicator {
  width: 28px;
  height: 28px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--primary-500);
  color: white;
  border-radius: 50%;
  flex-shrink: 0;
  animation: checkIn 0.3s cubic-bezier(0.4, 0, 0.2, 1);
  box-shadow: 0 2px 8px rgba(99, 102, 241, 0.3);
}

@keyframes checkIn {
  from {
    transform: scale(0) rotate(-45deg);
    opacity: 0;
  }
  to {
    transform: scale(1) rotate(0);
    opacity: 1;
  }
}
</style>