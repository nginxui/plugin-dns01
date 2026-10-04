// Vietnamese. Keys are the English source strings from DnsChallengeSlot.vue
// and the plugin settings schema in cmd/manifest. The DNS provider form
// phrases come from catalog/i18n, where cmd/manifest checks their coverage.
import providerPhrases from '../../../catalog/i18n/vi_VN.json'

export default {
  ...providerPhrases,
  'DNS Credential': 'Thông tin xác thực DNS',
  'Select Credential': 'Chọn thông tin xác thực',
  'Manage': 'Quản lý',
  'New credential': 'Thông tin xác thực mới',
  'Unknown Provider': 'Nhà cung cấp không xác định',
  'The previous credential was deleted. Select another one.': 'Thông tin xác thực đã chọn trước đó đã bị xóa. Hãy chọn thông tin khác.',
  'Advanced options': 'Tùy chọn nâng cao',
  'All default': 'Tất cả mặc định',
  'Follow CNAME': 'Theo CNAME',
  'Keep on when the validation record of a domain is delegated to another domain.': 'Giữ bật khi bản ghi xác thực của một tên miền được ủy quyền cho tên miền khác.',
  'CNAME following off': 'Đã tắt theo CNAME',
  'Check authoritative servers': 'Kiểm tra máy chủ có thẩm quyền',
  'When off, waits a fixed 60 seconds before asking for validation.': 'Khi tắt, sẽ chờ cố định 60 giây trước khi yêu cầu xác thực.',
  'Authoritative server check off': 'Đã tắt kiểm tra máy chủ có thẩm quyền',
  'Check public resolvers': 'Kiểm tra trình phân giải công cộng',
  'Confirms the record is visible through the recursive DNS servers set in the plugin settings.': 'Xác nhận bản ghi hiển thị qua các máy chủ DNS đệ quy đặt trong cài đặt plugin.',
  'Public resolver check off': 'Đã tắt kiểm tra trình phân giải công cộng',
  // Settings schema
  'Recursive DNS servers': 'Máy chủ DNS đệ quy',
  'Used to check that DNS records are visible. Empty means the system resolvers.': 'Dùng để kiểm tra bản ghi DNS đã hiển thị chưa. Để trống nghĩa là dùng trình phân giải của hệ thống.',
  'Default wait time (seconds)': 'Thời gian chờ mặc định (giây)',
  'How long to wait at most for a record to become visible when the provider does not say.': 'Thời gian chờ tối đa để bản ghi hiển thị khi nhà cung cấp không chỉ định.',
}
