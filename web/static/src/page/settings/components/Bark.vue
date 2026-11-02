<template>
  <div class="form-item">
    <label class="form-item-label">
      <span class="label-text">设备密钥</span>
    </label>
    <input
      v-model="modelValue.device_key"
      type="text"
      placeholder=""
      class="input input-bordered grow"
    />
  </div>
  <div class="form-item">
    <label class="form-item-label">
      <span class="label-text">Bark 服务器</span>
    </label>
    <input
      v-model="modelValue.bark_server"
      type="text"
      placeholder=""
      class="input input-bordered grow"
    />
  </div>
</template>

<script setup>
const props = defineProps({
  modelValue: {
    type: Object,
    required: true,
  },
});
</script>
