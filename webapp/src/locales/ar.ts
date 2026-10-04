// Arabic. Keys are the English source strings from DnsChallengeSlot.vue
// and the plugin settings schema in cmd/manifest. The DNS provider form
// phrases come from catalog/i18n, where cmd/manifest checks their coverage.
import providerPhrases from '../../../catalog/i18n/ar.json'

export default {
  ...providerPhrases,
  'DNS Credential': 'بيانات اعتماد DNS',
  'Select Credential': 'اختيار بيانات الاعتماد',
  'Manage': 'إدارة',
  'New credential': 'بيانات اعتماد جديدة',
  'Unknown Provider': 'مزود غير معروف',
  'The previous credential was deleted. Select another one.': 'حُذفت بيانات الاعتماد المختارة سابقًا. اختر بيانات أخرى.',
  'Advanced options': 'خيارات متقدمة',
  'All default': 'الكل افتراضي',
  'Follow CNAME': 'تتبّع CNAME',
  'Keep on when the validation record of a domain is delegated to another domain.': 'اتركه مفعّلًا عندما يكون سجل التحقق لنطاق ما مفوّضًا إلى نطاق آخر.',
  'CNAME following off': 'تتبّع CNAME متوقف',
  'Check authoritative servers': 'فحص الخوادم الموثوقة',
  'When off, waits a fixed 60 seconds before asking for validation.': 'عند إيقافه، ينتظر 60 ثانية ثابتة قبل طلب التحقق.',
  'Authoritative server check off': 'فحص الخوادم الموثوقة متوقف',
  'Check public resolvers': 'فحص المحللات العامة',
  'Confirms the record is visible through the recursive DNS servers set in the plugin settings.': 'يتأكد من ظهور السجل عبر خوادم DNS التكرارية المحددة في إعدادات الإضافة.',
  'Public resolver check off': 'فحص المحللات العامة متوقف',
  // Settings schema
  'Recursive DNS servers': 'خوادم DNS التكرارية',
  'Used to check that DNS records are visible. Empty means the system resolvers.': 'تُستخدم للتحقق من ظهور سجلات DNS. يعني الفراغ استخدام محللات النظام.',
  'Default wait time (seconds)': 'مدة الانتظار الافتراضية (بالثواني)',
  'How long to wait at most for a record to become visible when the provider does not say.': 'أقصى مدة انتظار لظهور السجل عندما لا يحددها المزود.',
}
