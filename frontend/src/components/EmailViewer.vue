<template>
  <div class="w-full p-5">
    <!-- Header -->
    <header class="mb-3 text-center">
      <h1 class="text-3xl font-bold text-gray-800">ENRON EMAILS SEARCHER</h1>
      <p class="text-gray-600">Search, filter and explore emails from the 150 employees involved in the Enron Company corruption scandal</p>
    </header>

    <!-- Email filters -->
    <div class="grid grid-cols-1 md:grid-cols-4 gap-4 bg-white p-4 shadow-lg rounded-lg">
      <input
        v-model="filters.query"
        @input="handleSearch"
        placeholder="Search by subject or content"
        class="w-full p-3 border border-gray-300 rounded-lg shadow-sm focus:ring-2 focus:ring-blue-500 focus:outline-none placeholder-gray-400"
      />
      <input
        v-model="filters.from"
        @input="handleSearch"
        placeholder="Filter by sender"
        class="w-full p-3 border border-gray-300 rounded-lg shadow-sm focus:ring-2 focus:ring-blue-500 focus:outline-none placeholder-gray-400"
      />
      <input
        v-model="filters.to"
        @input="handleSearch"
        placeholder="Filter by recipient"
        class="w-full p-3 border border-gray-300 rounded-lg shadow-sm focus:ring-2 focus:ring-blue-500 focus:outline-none placeholder-gray-400"
      />
      <input
        type="date"
        v-model="filters.datetime"
        @input="handleSearch"
        class="w-full p-3 border border-gray-300 rounded-lg shadow-sm focus:ring-2 focus:ring-blue-500 focus:outline-none placeholder-gray-400"
      />
    </div>

    <!-- Main content with table and email viewer -->
    <div class="flex flex-col md:flex-row gap-4 mt-4">
      <!-- Emails table -->
      <div
        class="flex-1 overflow-x-auto bg-white shadow-lg rounded-lg h-100 scrollable"
        :class="{ 'hidden md:block': selectedEmail && isMobile, 'block': !selectedEmail || !isMobile }"
      >
        <table class="min-w-full divide-y divide-gray-200">
          <thead class="bg-blue-500">
            <tr>
              <th
                v-for="header in tableHeaders"
                :key="header.key"
                @click="sortBy(header.key)"
                class="p-4 text-left text-white text-sm font-medium uppercase cursor-pointer hover:opacity-80"
              >
                {{ header.label }}
                <span v-if="sortConfig.key === header.key">
                  {{ sortConfig.direction === 'asc' ? '↑' : '↓' }}
                </span>
              </th>
            </tr>
          </thead>
          <tbody class="bg-white divide-y divide-gray-200 text-gray-700">
            <template v-if="emails.length > 0">
              <tr
                v-for="email in emails"
                :key="email._id"
                class="hover:bg-gray-100 transition"
              >
                <td
                  class="ps-4 pe-2 py-2 cursor-pointer text-blue-600 hover:text-blue-800"
                  @click="selectedEmail = email"
                >
                  {{ email.subject }}
                </td>
                <td class="ps-2 pe-2 py-2">
                  <div><span class="text-gray-400">from:</span> {{ email.from }}</div>
                  <div><span class="text-gray-400">To:</span> {{ email.to }}</div>
                </td>
                <td class="ps-2 pe-4 py-2">{{ formatDate(email.datetime) }}</td>
              </tr>
            </template>
            <tr v-else>
              <td colspan="4" class="p-8 text-center text-gray-500">
                <p class="text-lg font-medium">No se encontraron emails</p>
                <p class="text-sm mt-2">Intenta ajustar los filtros de búsqueda</p>
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      <!-- Email content viewer -->
      <div
        v-if="selectedEmail || !isMobile"
        class="w-full md:w-1/2 bg-white shadow-lg rounded-lg flex flex-col h-100"
        :class="{ 'hidden': !selectedEmail && isMobile }"
      >
        <template v-if="selectedEmail">
          <!-- Back button for mobile -->
          <button
            v-if="isMobile"
            @click="clearSelectedEmail"
            class="md:hidden m-4 px-4 py-2 bg-blue-500 text-white rounded-lg hover:bg-blue-600"
          >
            ← Back to List
          </button>

          <!-- Email header -->
          <div class="px-6 py-4 border-b">
            <h2 class="text-xl font-bold text-gray-800">{{ selectedEmail.subject }}</h2>
            <p class="text-sm text-gray-600 mt-1">
              From: {{ selectedEmail.from }}
              <br />
              To: {{ selectedEmail.to }}
              <br />
              Date: {{ formatDate(selectedEmail.datetime) }}
            </p>
          </div>

          <!-- Email body with scroll -->
          <div class="px-6 py-4 flex-1 overflow-y-auto scrollable">
            <p class="text-gray-700 whitespace-pre-wrap">{{ selectedEmail.body }}</p>
          </div>
        </template>
        <template v-else>
          <div class="h-full flex items-center justify-center text-gray-500">
            <p class="text-center">
              <span class="block text-lg font-medium">No message selected</span>
              <span class="block text-sm mt-2">Click on a message to view its content</span>
            </p>
          </div>
        </template>
      </div>
    </div>

    <!-- Pagination -->
    <div v-if="total > 0"
      class="flex justify-center items-center space-x-2 bg-white p-4 shadow-lg rounded-lg mt-4"
      :class="{ 'hidden md:flex': selectedEmail && isMobile, 'flex': !selectedEmail || !isMobile }"
    >
      <button
        @click="changePage(pagination.page - 1)"
        :disabled="pagination.page === 1"
        class="px-4 py-2 bg-blue-700 text-white rounded-lg shadow-md hover:bg-blue-600 disabled:bg-gray-300 disabled:cursor-not-allowed"
      >
        ⪡
      </button>
      <button
        v-for="(page, index) in getPageRange"
        :key="index"
        @click="typeof page === 'number' ? changePage(page) : null"
        :class="{
          'px-4 py-2 rounded-lg shadow-md': true,
          'bg-blue-500 text-white hover:bg-blue-600': typeof page === 'number' && page !== pagination.page,
          'bg-blue-600 text-white': page === pagination.page,
          'bg-transparent cursor-default': page === '...',
          'disabled:bg-gray-300 disabled:cursor-not-allowed': typeof page === 'number'
        }"
        :disabled="typeof page === 'number' && page === pagination.page"
      >
        {{ page }}
      </button>
      <button
        @click="changePage(pagination.page + 1)"
        :disabled="pagination.page === totalPages"
        class="px-4 py-2 bg-blue-700 text-white rounded-lg shadow-md hover:bg-blue-600 disabled:bg-gray-300 disabled:cursor-not-allowed"
      >
        ⪢
      </button>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted, onUnmounted, computed } from 'vue'

const emails = ref([])
const total = ref(0)
const totalPages = ref(0)
const selectedEmail = ref(null)
const isMobile = ref(false)

const tableHeaders = [
  { key: 'subject', label: 'Subject' },
  { key: 'from', label: 'Sender & Recipient' },
  { key: 'datetime', label: 'Date' }
]

const filters = reactive({
  query: '',
  from: '',
  to: '',
  datetime: ''
})

const sortConfig = reactive({
  key: 'datetime',
  direction: 'desc'
})

const pagination = reactive({
  page: 1,
  pageSize: 5
})

const updateIsMobile = () => {
  isMobile.value = window.innerWidth < 768
}

const clearSelectedEmail = () => {
  selectedEmail.value = null
}

const fetchEmails = async () => {
  try {
    const params = new URLSearchParams()

    // Add filters
    if (filters.query) params.append('query', filters.query)
    if (filters.from) params.append('from', filters.from)
    if (filters.to) params.append('to', filters.to)
    if (filters.datetime) params.append('dateTime', filters.datetime)

    // Add sorting
    params.append('sortBy', sortConfig.key)
    params.append('sortDir', sortConfig.direction)

    // Add pagination
    params.append('page', pagination.page)
    params.append('pageSize', pagination.pageSize)

    const response = await fetch(
      `http://localhost:3000/api/emails?${params.toString()}`
    )
    const data = await response.json()

    emails.value = data.emails
    total.value = data.total
    totalPages.value = Math.max(1, Math.ceil(data.total / pagination.pageSize))

    if (pagination.page > totalPages.value) {
      pagination.page = totalPages.value
      await fetchEmails()
    }
  } catch (error) {
    console.error('Error al obtener emails:', error)
    emails.value = []
    total.value = 0
    totalPages.value = 0
  }
}

const handleSearch = () => {
  pagination.page = 1
  fetchEmails()
}

const changePage = (newPage) => {
  if (newPage >= 1 && newPage <= totalPages.value) {
    pagination.page = newPage
    fetchEmails()
  }
}

const sortBy = (key) => {
  if (sortConfig.key === key) {
    sortConfig.direction = sortConfig.direction === 'asc' ? 'desc' : 'asc'
  } else {
    sortConfig.key = key
    sortConfig.direction = 'asc'
  }
  fetchEmails()
}

const formatDate = (datetime) => {
  const date = new Date(datetime);

  const day = String(date.getDate()).padStart(2, '0');
  const month = String(date.getMonth() + 1).padStart(2, '0');
  const year = date.getFullYear();

  const hour = String(date.getHours()).padStart(2, '0');
  const minute = String(date.getMinutes()).padStart(2, '0');

  return `${year}-${month}-${day} ${hour}:${minute}`;
}

const getPageRange = computed(() => {
  const current = pagination.page
  const last = totalPages.value
  const delta = 2
  const range = []
  const rangeWithDots = []
  let l

  range.push(1)

  for (let i = current - delta; i <= current + delta; i++) {
    if (i < last && i > 1) {
      range.push(i)
    }
  }

  for (let i of range) {
    if (l) {
      if (i - l === 2) {
        rangeWithDots.push(l + 1)
      } else if (i - l !== 1) {
        rangeWithDots.push('...')
      }
    }
    rangeWithDots.push(i)
    l = i
  }

  return rangeWithDots
})

onMounted(() => {
  updateIsMobile()
  window.addEventListener('resize', updateIsMobile)
  fetchEmails()
})

onUnmounted(() => {
  window.removeEventListener('resize', updateIsMobile)
})
</script>
